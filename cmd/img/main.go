package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"

	log "github.com/Sirupsen/logrus"
	"github.com/andrew-d/go-termutil"
	"github.com/mistifyio/lochness/pkg/hostport"
	"github.com/mistifyio/lochness/pkg/internal/cli"
	"github.com/mistifyio/mistify-image-service/metadata"
	logx "github.com/mistifyio/mistify-logrus-ext"
	"github.com/spf13/cobra"
)

var (
	server      = "images.service.lochness.local"
	jsonout     = false
	downloadDir = os.TempDir()
)

func help(cmd *cobra.Command, _ []string) {
	if err := cmd.Help(); err != nil {
		log.WithField("error", err).Fatal("help")
	}
}

func getImages(c *cli.Client) []cli.JMap {
	ret, _ := c.GetMany("images", "images")
	images := make([]cli.JMap, len(ret))
	for i := range ret {
		images[i] = ret[i]
	}
	return images
}

func list(cmd *cobra.Command, args []string) {
	c := cli.NewClient(getServerURL())
	images := []cli.JMap{}
	if len(args) == 0 {
		if termutil.Isatty(os.Stdin.Fd()) {
			images = getImages(c)
			sort.Sort(cli.JMapSlice(images))
		} else {
			args = cli.Read(os.Stdin)
		}
	}
	if len(images) == 0 {
		for _, id := range args {
			cli.AssertID(id)
			image, _ := c.Get("image", "images/"+id)
			images = append(images, image)
		}
	}

	for _, image := range images {
		image.Print(jsonout)
	}
}

func fetch(cmd *cobra.Command, specs []string) {
	c := cli.NewClient(getServerURL())
	if len(specs) == 0 {
		specs = cli.Read(os.Stdin)
	}
	for _, spec := range specs {
		cli.AssertSpec(spec)
		image, _ := c.Post("image", "images", spec)
		cli.JMap(image).Print(jsonout)
	}
}

func upload(cmd *cobra.Command, specs []string) {
	if len(specs) == 0 {
		specs = cli.Read(os.Stdin)
	}

	for _, spec := range specs {
		uploadImage := &metadata.Image{}
		if err := json.Unmarshal([]byte(spec), uploadImage); err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"json":  spec,
				"func":  "json.Unmarshal",
			}).Fatal("invalid spec")
		}

		sourcePath, err := filepath.Abs(uploadImage.Source)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"image": uploadImage,
				"func":  "filepath.Abs",
			}).Fatal("failed determine absolute source path")
		}
		file, err := os.Open(sourcePath)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"path":  sourcePath,
				"func":  "os.Open",
			}).Fatal("failed to open file")
		}
		defer file.Close()

		info, err := file.Stat()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"file":  file.Name(),
				"func":  "file.Stat",
			}).Fatal("failed to stat file")
		}

		uploadURL := getServerURL() + "/images"
		req, err := http.NewRequest("PUT", uploadURL, file)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"url":   uploadURL,
				"file":  file.Name(),
				"func":  "http.NewRequest",
			}).Fatal("failed to create request")
		}
		req.Header.Add("Content-Length", fmt.Sprintf("%d", info.Size()))
		req.Header.Add("X-Image-Type", uploadImage.Type)
		req.Header.Add("X-Image-Comment", uploadImage.Comment)
		req.Header.Add("Content-Type", "application/octet-stream")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"url":   uploadURL,
				"file":  file.Name(),
				"func":  "http.DefaultClient.Do",
			}).Fatal("request error")
		}
		image := &cli.JMap{}
		cli.ProcessResponse(res, "image", "upload", []int{http.StatusOK}, image)
		image.Print(jsonout)
	}
}

func download(cmd *cobra.Command, ids []string) {
	if len(ids) == 0 {
		ids = cli.Read(os.Stdin)
	}

	for _, id := range ids {
		success := false
		tempDest, err := ioutil.TempFile("", "incompleteImage-")
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"func":  "ioutil.TempFile",
			}).Fatal("could not create temporary file")
		}
		defer func() {
			if !success {
				if err := os.Remove(tempDest.Name()); err != nil {
					log.WithFields(log.Fields{
						"error":    err,
						"tempfile": tempDest.Name(),
						"func":     "os.Remove",
					}).Error("failed to remove temporary file")
				}
			}
		}()
		defer tempDest.Close()

		sourceURL := fmt.Sprintf("%s/images/%s/download", getServerURL(), id)
		resp, err := http.Get(sourceURL)
		if err != nil {
			log.WithFields(log.Fields{
				"error":     err,
				"sourceURL": sourceURL,
				"func":      "http.Get",
			}).Error("request error")
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.WithFields(log.Fields{
				"sourceURL":  sourceURL,
				"statusCode": resp.StatusCode,
				"func":       "http.Get",
			}).Error("bad response code")
			return
		}

		if _, err := io.Copy(tempDest, resp.Body); err != nil {
			log.WithFields(log.Fields{
				"error":     err,
				"sourceURL": sourceURL,
				"tempFile":  tempDest.Name(),
				"func":      "io.Copy",
			}).Error("failed to download image")
			return
		}

		if _, err := tempDest.Seek(0, 0); err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"tempFile": tempDest.Name(),
				"func":     "tempDest.Seek",
			}).Error("failed to seek to beginning of file")
		}
		fileBuffer := bufio.NewReader(tempDest)
		filetypeBytes, err := fileBuffer.Peek(512)
		if err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"tempFile": tempDest.Name(),
				"func":     "tempDest.Peek",
			}).Error("failed to read image filetype bytes")
			return
		}
		extension := ".tar"
		if http.DetectContentType(filetypeBytes) == "application/x-gzip" {
			extension = extension + ".gz"
		}

		imagePath := filepath.Join(downloadDir, id+extension)
		if err := os.Rename(tempDest.Name(), imagePath); err != nil {
			log.WithFields(log.Fields{
				"tempFile":  tempDest.Name(),
				"imagePath": imagePath,
				"func":      "os.Rename",
			}).Error("failed to rename image file")
			return
		}
		fmt.Println(imagePath)
		success = true
	}
}

func del(cmd *cobra.Command, ids []string) {
	c := cli.NewClient(getServerURL())
	if len(ids) == 0 {
		ids = cli.Read(os.Stdin)
	}

	for _, id := range ids {
		cli.AssertID(id)
		image, _ := c.Delete("image", "images/"+id)
		cli.JMap(image).Print(jsonout)
	}
}

func getServerURL() string {
	// Parse image service and do any necessary lookups
	host, port, err := hostport.Split(server)
	if err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"server": server,
			"func":   "hostport.Split",
		}).Fatal("host port split failed")
	}

	// Try to lookup port if only host/service is provided
	if port == "" {
		_, addrs, err := net.LookupSRV("", "", host)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"func":  "net.LookupSRV",
			}).Fatal("srv lookup failed")
		}
		if len(addrs) == 0 {
			log.WithField("server", host).Fatal("invalid host value")
		}
		port = fmt.Sprintf("%d", addrs[0].Port)
	}
	serverURL := &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, port),
	}
	return serverURL.String()
}

func main() {
	if err := logx.DefaultSetup("error"); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "logx.DefaultSetup",
			"level": "error",
		}).Fatal("unable to set up logrus")
	}

	root := &cobra.Command{
		Use:  "img",
		Long: "img is the command line interface to mistify-image-service. All commands support arguments via command line or stdin.",
		Run:  help,
	}
	root.PersistentFlags().BoolVarP(&jsonout, "json", "j", jsonout, "output in json")
	root.PersistentFlags().StringVarP(&server, "server", "s", server, "server address to connect to")

	cmdList := &cobra.Command{
		Use:   "list [<id>...]",
		Short: "List the images",
		Run:   list,
	}
	root.AddCommand(cmdList)

	cmdFetch := &cobra.Command{
		Use:   "fetch <spec>...",
		Short: "Fetch the image(s)",
		Long:  `Fetch new image(s) from a remote source. Where "spec" is a valid image metadata json string.`,
		Run:   fetch,
	}
	root.AddCommand(cmdFetch)

	cmdUpload := &cobra.Command{
		Use:   "upload <spec>...",
		Short: "Upload the image(s)",
		Long:  `Upload new image(s) from a local source. Where "spec" is a valid image metadata json string.`,
		Run:   upload,
	}
	root.AddCommand(cmdUpload)

	cmdDownload := &cobra.Command{
		Use:   "download <id>...",
		Short: "Download the image(s)",
		Run:   download,
	}
	cmdDownload.Flags().StringVarP(&downloadDir, "download-dir", "d", downloadDir, "directory to put downloaded image(s)")
	root.AddCommand(cmdDownload)

	cmdDelete := &cobra.Command{
		Use:   "delete <id>...",
		Short: "Delete images",
		Run:   del,
	}
	root.AddCommand(cmdDelete)

	if err := root.Execute(); err != nil {
		log.WithField("error", err).Fatal("failed to execute root command")
	}
}
