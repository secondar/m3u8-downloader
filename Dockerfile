FROM ubuntu:22.04
ENV DEBIAN_FRONTEND=noninteractive
RUN sed -i 's/archive.ubuntu.com/mirrors.aliyun.com/g' /etc/apt/sources.list && \
    sed -i 's/security.ubuntu.com/mirrors.aliyun.com/g' /etc/apt/sources.list
RUN apt-get update || apt-get update
RUN apt-get install -y \
    libgtk-3-0 \
    libnss3 \
    libcups2 \
    libxtst6 \
    libasound2 \
    libatk1.0-0 \
    libatk-bridge2.0-0 \
    libcairo2 \
    libdrm2 \
    libx11-6 \
    libxcomposite1 \
    libxdamage1 \
    libxext6 \
    libxfixes3 \
    libxrandr2 \
    libxrender1 \
    libxss1 \
    libxt6 \
    libgbm1
RUN mkdir -p /M3u8Download
COPY ./M3u8Download-Linux-x86_64 /M3u8Download/
COPY ./run.sh /M3u8Download/
COPY ./static /M3u8Download/static/
COPY ./data /M3u8Download/data/
WORKDIR /M3u8Download
RUN chmod +x /M3u8Download/M3u8Download-Linux-x86_64
EXPOSE 65533
CMD ["./M3u8Download-Linux-x86_64"]
