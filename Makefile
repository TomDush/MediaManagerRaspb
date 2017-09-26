# Replace demo with your desired executable name
appname := media-pi

sources := $(wildcard *.go)

build = GOOS=$(1) GOARCH=$(2) go build -o build/$(appname)$(3)
tar = cd build && tar -cvzf $(appname)-$(1)-$(2).tar.gz $(appname)$(3) VERSION media-pi.service && rm $(appname)$(3)
zip = cd build && zip $(appname)-$(1)-$(2).zip $(appname)$(3) VERSION media-pi.service && rm $(appname)$(3)

.PHONY: all windows darwin linux clean

all: mini
mini: build/media-pi-linux-amd64.tar.gz build/media-pi-linux-arm.tar.gz

clean:
	rm -rf build/ media-pi

version:
	mkdir -p build
	echo `date +'%Y%m%d%H%M%S'` > build/VERSION
	cp linux/media-pi.service build/

##### LINUX BUILDS #####
linux: build/media-pi-linux-arm.tar.gz build/media-pi-linux-arm64.tar.gz build/media-pi-linux-386.tar.gz build/media-pi-linux-amd64.tar.gz

build/media-pi-linux-386.tar.gz: $(sources) version
	$(call build,linux,386,)
	$(call tar,linux,386)

build/media-pi-linux-amd64.tar.gz: $(sources) version
	$(call build,linux,amd64,)
	$(call tar,linux,amd64)

build/media-pi-linux-arm.tar.gz: $(sources) version
	$(call build,linux,arm,)
	$(call tar,linux,arm)

build/media-pi-linux-arm64.tar.gz: $(sources) version
	$(call build,linux,arm64,)
	$(call tar,linux,arm64)

##### DARWIN (MAC) BUILDS #####
darwin: build/media-pi-darwin-amd64.tar.gz

build/media-pi-darwin-amd64.tar.gz: $(sources) version
	$(call build,darwin,amd64,)
	$(call tar,darwin,amd64)

##### WINDOWS BUILDS #####
windows: build/media-pi-windows-386.zip build/media-pi-windows-amd64.zip

build/media-pi-windows-386.zip: $(sources) version
	$(call build,windows,386,.exe)
	$(call zip,windows,386,.exe)

build/media-pi-windows-amd64.zip: $(sources) version
	$(call build,windows,amd64,.exe)
	$(call zip,windows,amd64,.exe)


publish: mini 
	rm -f build/VERSION build/media-pi.service
	aws s3 cp linux/PKGBUILD s3://dush-public/media-pi/PKGBUILD
	for bin in `ls build` ; do \
		aws s3 cp build/$$bin s3://dush-public/media-pi/$$bin ;\
	done
