FROM golang:1.18
COPY . /usr/src/app
CMD ["go", "run", "main.go"]
MAINTAINER juandreww108@gmail.com