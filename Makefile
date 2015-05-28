PREFIX := /usr
SBIN_DIR=$(PREFIX)/sbin

all: \
	cmd/cfirewalld/cfirewalld \
	cmd/enfield/enfield \
	cmd/dobharchu/dobharchu \
	cmd/cworkerd/cworkerd \
	cmd/heartbeat/heartbeat \
	cmd/nconfigd/nconfigd \
	cmd/loveland/loveland \
	cmd/waheela/waheela

cmd/cfirewalld/cfirewalld: cmd/cfirewalld/cfirewalld.go \
		cmd/cfirewalld/nftables.go
	cd $(dir $<) && \
	go get && \
	go build -v

$(SBIN_DIR)/cfirewalld: cmd/cfirewalld/cfirewalld
	install -D $< $(DESTDIR)$@


cmd/dobharchu/dobharchu: cmd/dobharchu/main.go
	cd $(dir $<) && \
	go get && \
	go build -v

$(SBIN_DIR)/dobharchu: cmd/dobharchu/dobharchu
	install -D $< $(DESTDIR)$@


cmd/cworkerd/cworkerd: cmd/cworkerd/main.go
	cd $(dir $<) && \
	go get && \
	go build -v

$(SBIN_DIR)/cworkerd: cmd/cworkerd/cworkerd
	install -D $< $(DESTDIR)$@


cmd/enfield/enfield: cmd/enfield/main.go
	cd $(dir $<) && \
	go get && \
	go build -v

$(SBIN_DIR)/enfield: cmd/enfield/enfield
	install -D $< $(DESTDIR)$@


cmd/chypervisord/chypervisord: cmd/chypervisord/main.go \
		cmd/chypervisord/helpers.go \
		cmd/chypervisord/http.go \
		cmd/chypervisord/hypervisor.go
	cd $(dir $<) && \
	go get && \
	go build -v

$(SBIN_DIR)/chypervisord: cmd/chypervisord/chypervisord
	install -D $< $(DESTDIR)$@


cmd/heartbeat/heartbeat: cmd/heartbeat/main.go
	cd $(dir $<) && \
	go get && \
	go build -v

$(SBIN_DIR)/heartbeat: cmd/heartbeat/heartbeat
	install -D $< $(DESTDIR)$@


cmd/nconfigd/nconfigd: cmd/nconfigd/main.go
	cd $(dir $<) && \
	go get && \
	go build -v

$(SBIN_DIR)/nconfigd: cmd/nconfigd/nconfigd
	install -D $< $(DESTDIR)$@


cmd/loveland/loveland: cmd/loveland/main.go
	cd $(dir $<) && \
	go get && \
	go build -v

$(SBIN_DIR)/loveland: cmd/loveland/loveland
	install -D $< $(DESTDIR)$@


cmd/waheela/waheela: cmd/waheela/main.go \
		cmd/waheela/guest.go \
		cmd/waheela/helpers.go \
		cmd/waheela/http.go
	cd $(dir $<) && \
	go get && \
	go build -v

$(SBIN_DIR)/waheela: cmd/waheela/waheela
	install -D $< $(DESTDIR)$@


clean:
	cd cmd/cfirewalld && \
	go clean -x

	cd cmd/dobharchu && \
	go clean

	cd cmd/cworkerd && \
	go clean

	cd cmd/enfield && \
	go clean

	cd cmd/chypervisord && \
	go clean

	cd cmd/heartbeat && \
	go clean

	cd cmd/nconfigd && \
	go clean

	cd cmd/loveland && \
	go clean -x

	cd cmd/waheela && \
	go clean -x


install: \
  $(SBIN_DIR)/cfirewalld \
  $(SBIN_DIR)/dobharchu \
  $(SBIN_DIR)/cworkerd \
  $(SBIN_DIR)/enfield \
  $(SBIN_DIR)/chypervisord \
  $(SBIN_DIR)/heartbeat \
  $(SBIN_DIR)/nconfigd \
  $(SBIN_DIR)/loveland \
  $(SBIN_DIR)/waheela \

