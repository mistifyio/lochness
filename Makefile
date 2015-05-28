PREFIX := /usr
SBIN_DIR=$(PREFIX)/sbin

all: \
	cmd/cfirewalld/cfirewalld \
	cmd/enfield/enfield \
	cmd/cdhcpd/cdhcpd \
	cmd/cworkerd/cworkerd \
	cmd/nheartbeatd/nheartbeatd \
	cmd/nconfigd/nconfigd \
	cmd/loveland/loveland \
	cmd/cguestd/cguestd

cmd/cfirewalld/cfirewalld: cmd/cfirewalld/cfirewalld.go \
		cmd/cfirewalld/nftables.go
	cd $(dir $<) && \
	go get && \
	go build -v

$(SBIN_DIR)/cfirewalld: cmd/cfirewalld/cfirewalld
	install -D $< $(DESTDIR)$@


cmd/cdhcpd/cdhcpd: cmd/cdhcpd/main.go
	cd $(dir $<) && \
	go get && \
	go build -v

$(SBIN_DIR)/cdhcpd: cmd/cdhcpd/cdhcpd
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


cmd/nheartbeatd/nheartbeatd: cmd/nheartbeatd/main.go
	cd $(dir $<) && \
	go get && \
	go build -v

$(SBIN_DIR)/nheartbeatd: cmd/nheartbeatd/nheartbeatd
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


cmd/cguestd/cguestd: cmd/cguestd/main.go \
		cmd/cguestd/guest.go \
		cmd/cguestd/helpers.go \
		cmd/cguestd/http.go
	cd $(dir $<) && \
	go get && \
	go build -v

$(SBIN_DIR)/cguestd: cmd/cguestd/cguestd
	install -D $< $(DESTDIR)$@


clean:
	cd cmd/cfirewalld && \
	go clean -x

	cd cmd/cdhcpd && \
	go clean

	cd cmd/cworkerd && \
	go clean

	cd cmd/enfield && \
	go clean

	cd cmd/chypervisord && \
	go clean

	cd cmd/nheartbeatd && \
	go clean

	cd cmd/nconfigd && \
	go clean

	cd cmd/loveland && \
	go clean -x

	cd cmd/cguestd && \
	go clean -x


install: \
  $(SBIN_DIR)/cfirewalld \
  $(SBIN_DIR)/cdhcpd \
  $(SBIN_DIR)/cworkerd \
  $(SBIN_DIR)/enfield \
  $(SBIN_DIR)/chypervisord \
  $(SBIN_DIR)/nheartbeatd \
  $(SBIN_DIR)/nconfigd \
  $(SBIN_DIR)/loveland \
  $(SBIN_DIR)/cguestd \

