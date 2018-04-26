# Cross platform GOLANG project build
appname := medima-pi

sources := $(wildcard *.go)

build = GOOS=$(1) GOARCH=$(2) go build -o build/$(appname)$(3)
tar = cd build && tar -cvzf $(appname)-$(1)-$(2).tar.gz $(appname)$(3) VERSION medima-pi.service && rm $(appname)$(3)
zip = cd build && zip $(appname)-$(1)-$(2).zip $(appname)$(3) VERSION medima-pi.service && rm $(appname)$(3)

.PHONY: all windows darwin linux clean

DEST?=dush@192.168.0.11:~/medima

all: mini
mini: build/medima-pi-linux-amd64.tar.gz build/medima-pi-linux-arm.tar.gz

clean:
	rm -rf build/ medima-pi

test:
	go test

version: $(sources)
	mkdir -p build
	echo `date +'%Y%m%d%H%M%S'` > build/VERSION
	cp linux/medima-pi.service build/

snapshot: all
	scp build/medima-pi-linux-arm.tar.gz $(DEST)

##### LINUX BUILDS #####
linux: build/medima-pi-linux-arm.tar.gz build/medima-pi-linux-arm64.tar.gz build/medima-pi-linux-386.tar.gz build/medima-pi-linux-amd64.tar.gz

build/medima-pi-linux-386.tar.gz: $(sources) version test
	$(call build,linux,386,)
	$(call tar,linux,386)

build/medima-pi-linux-amd64.tar.gz: $(sources) version test
	$(call build,linux,amd64,)
	$(call tar,linux,amd64)

build/medima-pi-linux-arm.tar.gz: $(sources) version test
	$(call build,linux,arm,)
	$(call tar,linux,arm)

build/medima-pi-linux-arm64.tar.gz: $(sources) version test
	$(call build,linux,arm64,)
	$(call tar,linux,arm64)

##### DARWIN (MAC) BUILDS #####
darwin: build/medima-pi-darwin-amd64.tar.gz

build/medima-pi-darwin-amd64.tar.gz: $(sources) version test
	$(call build,darwin,amd64,)
	$(call tar,darwin,amd64)

##### WINDOWS BUILDS #####
windows: build/medima-pi-windows-386.zip build/medima-pi-windows-amd64.zip

build/medima-pi-windows-386.zip: $(sources) version test
	$(call build,windows,386,.exe)
	$(call zip,windows,386,.exe)

build/medima-pi-windows-amd64.zip: $(sources) version test
	$(call build,windows,amd64,.exe)
	$(call zip,windows,amd64,.exe)


publish: mini 
	rm -f build/VERSION build/medima-pi.service
	aws s3 cp linux/PKGBUILD s3://dush-public/medima-pi/PKGBUILD
	for bin in `ls build` ; do \
		aws s3 cp build/$$bin s3://dush-public/medima-pi/$$bin ;\
	done