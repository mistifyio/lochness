# img

[![img](https://godoc.org/github.com/mistifyio/lochness/cmd/img?status.png)](https://godoc.org/github.com/mistifyio/lochness/cmd/img)

img is the command line interface to mistify-image-service, the image management
service.


### Usage

The following arguments are understood

    $ img -h
    img is the command line interface to mistify-image-service. All commands support arguments via command line or stdin.

    Usage:
    img [flags]
    img [command]

    Available Commands:
    list        List the images
    fetch       Fetch the image(s)
    upload      Upload the image(s)
    download    Download the image(s)
    delete      Delete images
    help        Help about any command

    Flags:
    -h, --help=false: help for img
    -j, --json=false: output in json
    -s, --server="image.services.lochness.local": server address to connect to

    Use "img help [command]" for more information about a command.

Input is supported via command line or stdin.


### Output

All commands except download support two output formats, a list of image ids or
a list of JSON objects, line separated. The JSON is a metadata.Image object.


### Examples

List images

    $ img list
    27925fad-2243-4dd3-99e1-ea5f5df33c6b
    95f012e0-56a5-47e0-96df-38b806feda63

    $ img list -j
    list -j
    {"comment":"foo2","download_end":"2015-06-08T18:08:39.715868326Z","download_start":"2015-06-08T18:08:39.709827698Z","expected_size":-1,"id":"27925fad-2243-4dd3-99e1-ea5f5df33c6b","size":626965,"source":"","status":"complete","type":"container"}
    {"comment":"","download_end":"2015-06-08T18:08:39.703626291Z","download_start":"2015-06-08T18:08:39.69532952Z","expected_size":-1,"id":"95f012e0-56a5-47e0-96df-38b806feda63","size":626965,"source":"","status":"complete","type":"container"}

    $img list -j 95f012e0-56a5-47e0-96df-38b806feda63
    {"comment":"","download_end":"2015-06-08T18:08:39.703626291Z","download_start":"2015-06-08T18:08:39.69532952Z","expected_size":-1,"id":"95f012e0-56a5-47e0-96df-38b806feda63","size":626965,"source":"","status":"complete","type":"container"}

Upload image

    $ img upload '{"type":"container", "source":"foo.tar.gz"}' '{"type":"container", "comment": "foo2", "source":"foo2.tar.gz"}'
    95f012e0-56a5-47e0-96df-38b806feda63
    27925fad-2243-4dd3-99e1-ea5f5df33c6b

    $ img upload '{"type":"container", "source":"foo.tar.gz"}' '{"type":"container", "comment": "foo2", "source":"foo2.tar.gz"}'
    {"comment":"","download_end":"2015-06-08T18:09:04.601465793Z","download_start":"2015-06-08T18:09:04.595228495Z","expected_size":-1,"id":"95f012e0-56a5-47e0-96df-38b806feda63","size":626965,"source":"","status":"complete","type":"container"}
    {"comment":"foo2","download_end":"2015-06-08T18:09:04.61605015Z","download_start":"2015-06-08T18:09:04.6084636Z","expected_size":-1,"id":"27925fad-2243-4dd3-99e1-ea5f5df33c6b","size":626965,"source":"","status":"complete","type":"container"}

Fetch image

    $ img fetch '{"type":"container","source":"http://localhost:20000/images/9848b0a2-1e49-49e3-b9f1-ad2b9f2b509d/download"}'
    67601793-7c27-4fa1-beb4-4d93de437b38

Download image

    $ img download -d /tmp 95f012e0-56a5-47e0-96df-38b806feda63
    /tmp/95f012e0-56a5-47e0-96df-38b806feda63.tar.gz

Delete image

    $ img delete 95f012e0-56a5-47e0-96df-38b806feda63
    95f012e0-56a5-47e0-96df-38b806feda63


--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*
