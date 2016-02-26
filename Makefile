rwildcard=$(foreach d,$(wildcard $1*),$(call rwildcard,$d/,$2) $(filter $(subst *,%,$2),$d))

SHELL := /bin/bash

PREFIX := /usr
SBIN_DIR=$(PREFIX)/sbin
CMDS :=  \
	cbootstrapd \
	cdhcpd \
	cguestd \
	chypervisord \
	cnetworkd \
	cplacerd \
	cworkerd \
	guest \
	hv \
	img \
	lock \
	locker \
	nconfigd \
	nfirewalld \
	nheartbeatd \


test_files := $(call rwildcard,,*_test.go)
pkgdirs := $(filter-out ./, $(sort $(dir $(test_files))))
pkgs := $(notdir $(patsubst %/,%,$(pkgdirs)))
tests := $(join $(pkgdirs), $(addsuffix .test,$(pkgs))) lochness.test

BINS := $(join $(addprefix cmd/,$(CMDS)) ,$(addprefix /,$(CMDS)))
all: $(BINS)

 $(BINS):
	@echo BUILD $@
	@cd $(dir $<) && go build

cmd/cbootstrapd/cbootstrapd: $(wildcard cmd/cbootstrapd/*.go)
cmd/cdhcpd/cdhcpd: $(wildcard cmd/cdhcpd/*.go)
cmd/cguestd/cguestd: $(wildcard cmd/cguestd/*.go)
cmd/chypervisord/chypervisord: $(wildcard cmd/chypervisord/*.go)
cmd/cnetworkd/cnetworkd: $(wildcard cmd/cnetworkd/*.go)
cmd/cplacerd/cplacerd: $(wildcard cmd/cplacerd/*.go)
cmd/cworkerd/cworkerd: $(wildcard cmd/cworkerd/*.go)
cmd/guest/guest: $(wildcard cmd/guest/*.go)
cmd/hv/hv: $(wildcard cmd/hv/*.go)
cmd/img/img: $(wildcard cmd/img/*.go)
cmd/lock/lock: $(wildcard cmd/lock/*.go)
cmd/locker/locker: $(wildcard cmd/locker/*.go)
cmd/nconfigd/nconfigd: $(wildcard cmd/nconfigd/*.go)
cmd/nfirewalld/nfirewalld: $(wildcard cmd/nfirewalld/*.go)
cmd/nheartbeatd/nheartbeatd: $(wildcard cmd/nheartbeatd/*.go)

internal/cli/cli.test: $(wildcard internal/cli/*.go)
pkg/deferer/deferer.test: $(wildcard pkg/deferer/*.go)
pkg/jobqueue/jobqueue.test: $(wildcard pkg/jobqueue/*.go)
pkg/lock/lock.test: $(wildcard pkg/lock/*.go)
pkg/sd/sd.test: $(wildcard pkg/sd/*.go)
pkg/watcher/watcher.test: $(wildcard pkg/watcher/*.go)

$(SBIN_DIR)/%:
	install -D $< $(DESTDIR)$@

$(SBIN_DIR)/cbootstrapd: cmd/cbootstrapd/cbootstrapd
$(SBIN_DIR)/cdhcpd: cmd/cdhcpd/cdhcpd
$(SBIN_DIR)/cguestd: cmd/cguestd/cguestd
$(SBIN_DIR)/chypervisord: cmd/chypervisord/chypervisord
$(SBIN_DIR)/cnetworkd: cmd/cnetworkd/cnetworkd
$(SBIN_DIR)/cplacerd: cmd/cplacerd/cplacerd
$(SBIN_DIR)/cworkerd: cmd/cworkerd/cworkerd
$(SBIN_DIR)/lock: cmd/lock/lock
$(SBIN_DIR)/locker: cmd/locker/locker
$(SBIN_DIR)/nconfigd: cmd/nconfigd/nconfigd
$(SBIN_DIR)/nfirewalld: cmd/nfirewalld/nfirewalld
$(SBIN_DIR)/nheartbeatd: cmd/nheartbeatd/nheartbeatd

.PHONY: test
test: $(addsuffix .run,$(tests))
	@cat $(addsuffix .out,$(tests))

.PHONY: %.test.run
%.test.run: %.test
	@echo "TEST  $^"
	@cid=$(shell docker run -dti -v "${PWD}:/lochness:ro" -v /sys/fs/cgroup:/sys/fs/cgroup:ro --name $(notdir $^) mistifyio/mistify-os) && \
	test -n $(cid) && \
	sleep .25 && \
	docker exec $$cid sh -c "cd /lochness; cd $(@D); LOCHNESS_TEST_NO_BUILD=1 ./$(notdir $^) -test.v" &> $^.out; \
	docker kill $$cid &>/dev/null && \
	docker rm -v $$cid >& /dev/null

.SECONDARY: $(tests)
%.test: %
	@echo BUILD $@
	@cd $(dir $^) && go test -c

lochness.test:
	@echo BUILD $@
	@go test -c

clean:
	for d in $(dir $(CMDS)); do (cd $$d && go clean); done


install: $(addprefix $(SBIN_DIR)/,$(filter-out guest hv img,$(CMDS)))
