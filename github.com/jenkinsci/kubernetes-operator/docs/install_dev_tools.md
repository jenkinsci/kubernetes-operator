# Installation of the required development tools

This document explains how to install the Go tools used by the development process.

## Configure environment variables

```bash
export GOPATH=/home/go # example value
export GOROOT=/usr/lib/go-1.12 # example value
export PATH=$GOPATH/bin:$PATH
```

## goimports

```
go get golang.org/x/tools/cmd/goimports
cd $GOPATH/src/golang.org/x/tools/cmd/goimports
go build
go install
```

## golint

```
go get -u golang.org/x/lint/golint
cd  $GOPATH/src/golang.org/x/lint/golint
go build
go install
```

## checkmake
```
go get github.com/mrtazz/checkmake
cd $GOPATH/src/github.com/mrtazz/checkmake
go build
go install
```

## staticcheck

```
mkdir -p $GOPATH/src/github.com/dominikh/
cd $GOPATH/src/github.com/dominikh/
git clone https://github.com/dominikh/go-tools.git
cd  $GOPATH/src/github.com/dominikh/go-tools/staticcheck
go build
go install
```


