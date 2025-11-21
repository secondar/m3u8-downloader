FROM ubuntu:22.04
ENV DEBIAN_FRONTEND=noninteractive
RUN mkdir -p /M3u8Download
COPY ./M3u8Download-Linux-x86_64 /M3u8Download/
COPY ./run.sh /M3u8Download/
COPY ./static /M3u8Download/static/
COPY ./data /M3u8Download/data/
WORKDIR /M3u8Download
RUN chmod +x /M3u8Download/M3u8Download-Linux-x86_64
EXPOSE 65533
CMD ["./M3u8Download-Linux-x86_64"]
