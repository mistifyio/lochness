PREFIX := /usr
SBIN_DIR=$(PREFIX)/sbin

all: \
	cmd/nfirewalld/nfirewalld \
	cmd/cbootstrapd/cbootstrapd \
	cmd/cdhcpd/cdhcpd \
	cmd/cworkerd/cworkerd \
	cmd/nheartbeatd/nheartbeatd \
	cmd/nconfigd/nconfigd \
	cmd/cplacerd/cplacerd \
	cmd/cguestd/cguestd

cmd/nfirewalld/nfirewalld: cmd/nfirewalld/nfirewalld.go \
		cmd/nfirewalld/nftables.go
	cd $(dir $<) && \
	go get && \
	go build -v

$(SBIN_DIR)/nfirewalld: cmd/nfirewalld/nfirewalld
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


cmd/cbootstrapd/cbootstrapd: cmd/cbootstrapd/main.go
	cd $(dir $<) && \
	go get && \
	go build -v

$(SBIN_DIR)/cbootstrapd: cmd/cbootstrapd/cbootstrapd
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


cmd/cplacerd/cplacerd: cmd/cplacerd/main.go
	cd $(dir $<) && \
	go get && \
	go build -v

$(SBIN_DIR)/cplacerd: cmd/cplacerd/cplacerd
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

cmd/cnetworkd/cnetworkd: cmd/cnetworkd/main.go \
		cmd/cnetworkd/vlan.go \
		cmd/cnetworkd/vlangroup.go \
		cmd/cnetworkd/helpers.go \
		cmd/cnetworkd/http.go
	cd $(dir $<) && \
	go get && \
	go build -v

$(SBIN_DIR)/cnetworkd: cmd/cnetworkd/cnetworkd
	install -D $< $(DESTDIR)$@

clean:
	cd cmd/nfirewalld && \
	go clean -x

	cd cmd/cdhcpd && \
	go clean

	cd cmd/cworkerd && \
	go clean

	cd cmd/cbootstrapd && \
	go clean

	cd cmd/chypervisord && \
	go clean

	cd cmd/nheartbeatd && \
	go clean

	cd cmd/nconfigd && \
	go clean

	cd cmd/cplacerd && \
	go clean -x

	cd cmd/cguestd && \
	go clean -x

	cd cmd/cnetworkd && \
	go clean -x


install: \
  $(SBIN_DIR)/nfirewalld \
  $(SBIN_DIR)/cdhcpd \
  $(SBIN_DIR)/cworkerd \
  $(SBIN_DIR)/cbootstrapd \
  $(SBIN_DIR)/chypervisord \
  $(SBIN_DIR)/nheartbeatd \
  $(SBIN_DIR)/nconfigd \
  $(SBIN_DIR)/cplacerd \
  $(SBIN_DIR)/cguestd \
  $(SBIN_DIR)/cnetworkd \

