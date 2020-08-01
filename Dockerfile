FROM golang as builder

RUN mkdir /sandman
WORKDIR /sandman
COPY ./ /sandman/

RUN go get "github.com/fasthttp/websocket"
RUN go get "github.com/valyala/fasthttp"
RUN go get "github.com/valyala/fasthttp/prefork"
RUN go build
RUN cp /sandman/sandman /usr/local/bin

FROM python:3
COPY --from=builder /usr/local/bin/sandman /usr/local/bin/sandman
RUN mkdir /sandman
WORKDIR /sandman
#RUN git clone https://github.com/ChillFish8/GoFasterSandman.git ./modules/sandman
#RUN cp ./modules/sandman/requirements.txt .
COPY ./requirements.txt /sandman/
RUN pip install -r ./requirements.txt