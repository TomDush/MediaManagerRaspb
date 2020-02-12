# Media PI

[![CircleCI](https://circleci.com/gh/TomDush/medima-pi.svg?style=svg)](https://circleci.com/gh/TomDush/medima-pi)

Simple media manager for Raspberry PI 3 developed in GOLANG.

## Installation on Archlinux

Using Pacman:

    rm -rf medima-pi && mkdir medima-pi && cd medima-pi
    wget https://s3-eu-west-1.amazonaws.com/dush-public/medima-pi/PKGBUILD
    makepkg -sri

    system-ctl enable medima-pi
    system-ctl start medima-pi

## Development Environment

Install required tools:

    sudo apt install golang go-dep
    
Clone project:

    go get github.com/TomDush/medima-pi
    cd "$GOPATH"/src/github.com/TomDush/medima-pi
    dep ensure
   
Run:

    # run server locally
    ./grun
    
    # run unit-tests
    go test

## Build & publish

Build and publish (s3, aws cmd must be configured):

    make publish


