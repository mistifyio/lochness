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
testBins := $(addsuffix .test,$(pkgs))
tests := $(join $(pkgdirs), $(testBins)) lochness.test

BINS := $(join $(addprefix cmd/,$(CMDS)) ,$(addprefix /,$(CMDS)))
all: $(BINS)

.SILENT:

$(BINS):
	echo BUILD $@
	cd $(dir $<) && go build

pkgs := $(call rwildcard,pkg,*.go)

$(tests): $(wildcard internal/tests/common/*.go)
cmd/cbootstrapd/cbootstrapd cmd/cbootstrapd/cbootstrapd.test: $(wildcard cmd/cbootstrapd/*.go) $(pkgs)
cmd/cdhcpd/cdhcpd cmd/cdhcpd/cdhcpd.test: $(wildcard cmd/cdhcpd/*.go) $(pkgs)
cmd/cguestd/cguestd cmd/cguestd/cguestd.test: $(wildcard cmd/cguestd/*.go) $(pkgs)
cmd/chypervisord/chypervisord cmd/chypervisord/chypervisord.test: $(wildcard cmd/chypervisord/*.go) $(pkgs)
cmd/cnetworkd/cnetworkd cmd/cnetworkd/cnetworkd.test: $(wildcard cmd/cnetworkd/*.go) $(pkgs)
cmd/cplacerd/cplacerd cmd/cplacerd/cplacerd.test: $(wildcard cmd/cplacerd/*.go) $(pkgs)
cmd/cworkerd/cworkerd cmd/cworkerd/cworkerd.test: $(wildcard cmd/cworkerd/*.go) $(pkgs)
cmd/guest/guest cmd/guest/guest.test: $(wildcard cmd/guest/*.go) $(pkgs)
cmd/hv/hv cmd/hv/hv.test: $(wildcard cmd/hv/*.go) $(pkgs)
cmd/img/img cmd/img/img.test: $(wildcard cmd/img/*.go) $(pkgs)
cmd/lock/lock cmd/lock/lock.test: $(wildcard cmd/lock/*.go) $(pkgs)
cmd/locker/locker cmd/locker/locker.test: $(wildcard cmd/locker/*.go) $(pkgs)
cmd/nconfigd/nconfigd cmd/nconfigd/nconfigd.test: $(wildcard cmd/nconfigd/*.go) $(pkgs)
cmd/nfirewalld/nfirewalld cmd/nfirewalld/nfirewalld.test: $(wildcard cmd/nfirewalld/*.go) $(pkgs)
cmd/nheartbeatd/nheartbeatd cmd/nheartbeatd/nheartbeatd.test: $(wildcard cmd/nheartbeatd/*.go) $(pkgs)

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
test: $(addsuffix .run.out,$(tests))

FORCE:

.PHONY: %.test.run.out
%.test.run.out: %.test.run FORCE
	flock /dev/stdout -c 'cat $@'

.PHONY: %.test.run
%.test.run: %.test %
	flock /dev/stdout -c 'echo "RUN   $<"'
	cid=$(shell docker run -dti -v "$(CURDIR):/lochness:ro" -v /sys/fs/cgroup:/sys/fs/cgroup:ro --name $(notdir $<) mistifyio/mistify-os) && \
	test -n $(cid) && \
	sleep .25 && \
	docker exec $$cid sh -c "cd /lochness; cd $(@D); LOCHNESS_TEST_NO_BUILD=1 ./$(notdir $<) -test.v" &> $@.out; \
	ret=$$?; \
	docker kill $$cid  &>/dev/null && \
	docker rm -v $$cid &>/dev/null && \
	flock /dev/stdout -c 'echo "+++ $< +++"; cat $@.out'; \
	exit $$ret

.SECONDARY: $(tests)
%.test:
	echo BUILD $@
	cd $(dir $@) && flock -s /dev/stdout go test -c

.PHONY: lochness
lochness.test:
	echo BUILD $@
	go test -c

.PHONY: internal/tests/common internal/tests/common.test
internal/tests/common internal/tests/common.test:

.PHONY: internal/cli/cli pkg/deferer/deferer pkg/jobqueue/jobqueue pkg/lock/lock pkg/sd/sd pkg/watcher/watcher
internal/cli/cli.test: $(wildcard internal/cli/*.go)
pkg/deferer/deferer.test pkg/deferer/deferer.test: $(wildcard pkg/deferer/*.go)
pkg/jobqueue/jobqueue.test: $(wildcard pkg/jobqueue/*.go)
pkg/lock/lock.test: $(wildcard pkg/lock/*.go)
pkg/sd/sd.test: $(wildcard pkg/sd/*.go)
pkg/watcher/watcher.test: $(wildcard pkg/watcher/*.go)

clean:
	for d in $(dir $(CMDS)); do (cd $$d && go clean); done


install: $(addprefix $(SBIN_DIR)/,$(filter-out guest hv img,$(CMDS)))
