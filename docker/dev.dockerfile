FROM golang:alpine

# Alpine images doesn't include base tools
# so install necessary tools
RUN apk update && apk upgrade \
  && apk add --no-cache git make

WORKDIR /idp

# Copy go modules
COPY go.mod go.sum ./

# Download project deps
# Add air for hot reload
RUN go mod download \
  && go get -u -v github.com/cosmtrek/air@master 

# Copy the rest of the app to container
COPY . .

EXPOSE 8080

CMD [ "air", "-c", ".air.toml" ]
