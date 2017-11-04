# Media PI

Simple media manager for Raspberry PI 3 developed in GOLANG.

## Installation

On Archlinux, download https://s3-eu-west-1.amazonaws.com/dush-public/medima-pi/PKGBUILD, then run:

    rm -rf medima-pi && mkdir medima-pi && cd medima-pi
    wget https://s3-eu-west-1.amazonaws.com/dush-public/medima-pi/PKGBUILD
    makepkg -sri

    system-ctl enable medima-pi
    system-ctl start medima-pi

## Development

### Clone this repo

    go get github.com/TomDush/medima-pi

### Build & publish

Test locally

    ./grun

Build and publish (s3, aws cmd must be configured):

    make publish


