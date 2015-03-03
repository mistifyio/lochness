PREFIX := /usr
SBIN_DIR=$(PREFIX)/sbin

all: \
	cmd/cherufe/cherufe \
	cmd/enfield/enfield \
	cmd/dover/dover \
	cmd/heartbeat/heartbeat \
	cmd/kappa/kappa \
	cmd/loveland/loveland \
	cmd/waheela/waheela

cmd/cherufe/cherufe: cmd/cherufe/cherufe.go \
		cmd/cherufe/nftables.go
	cd $(dir $<) && \
	go get && \
	go build -v

$(SBIN_DIR)/cherufe: cmd/cherufe/cherufe
	install -D $< $(DESTDIR)$@


cmd/dover/dover: cmd/dover/main.go
	cd $(dir $<) && \
	go get && \
	go build -v

$(SBIN_DIR)/dover: cmd/dover/dover
	install -D $< $(DESTDIR)$@


cmd/enfield/enfield: cmd/enfield/main.go
	cd $(dir $<) && \
	go get && \
	go build -v

$(SBIN_DIR)/enfield: cmd/enfield/enfield
	install -D $< $(DESTDIR)$@


cmd/grootslang/grootslang: cmd/grootslang/main.go \
		cmd/grootslang/helpers.go \
		cmd/grootslang/http.go \
		cmd/grootslang/hypervisor.go
	cd $(dir $<) && \
	go get && \
	go build -v

$(SBIN_DIR)/grootslang: cmd/grootslang/grootslang
	install -D $< $(DESTDIR)$@


cmd/heartbeat/heartbeat: cmd/heartbeat/main.go
	cd $(dir $<) && \
	go get && \
	go build -v

$(SBIN_DIR)/heartbeat: cmd/heartbeat/heartbeat
	install -D $< $(DESTDIR)$@


cmd/kappa/kappa: cmd/kappa/main.go
	cd $(dir $<) && \
	go get && \
	go build -v

$(SBIN_DIR)/kappa: cmd/kappa/kappa
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
	cd cmd/cherufe && \
	go clean -x

	cd cmd/dover && \
	go clean

	cd cmd/enfield && \
	go clean

	cd cmd/grootslang && \
	go clean

	cd cmd/heartbeat && \
	go clean

	cd cmd/kappa && \
	go clean

	cd cmd/loveland && \
	go clean -x

	cd cmd/waheela && \
	go clean -x


install: \
  $(SBIN_DIR)/cherufe \
  $(SBIN_DIR)/dover \
  $(SBIN_DIR)/enfield \
  $(SBIN_DIR)/grootslang \
  $(SBIN_DIR)/heartbeat \
  $(SBIN_DIR)/kappa \
  $(SBIN_DIR)/loveland \
  $(SBIN_DIR)/waheela \

