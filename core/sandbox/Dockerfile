FROM alpine:3.21

RUN apk add --no-cache \
    bash curl git vim wget sudo \
    go nodejs npm python3 py3-pip \
    tmux htop openssh-client docker-cli \
    openssl ca-certificates tzdata

# ttyd web terminal
RUN wget -qO /usr/local/bin/ttyd https://github.com/tsl0922/ttyd/releases/download/1.7.7/ttyd.x86_64 && \
    chmod +x /usr/local/bin/ttyd

RUN adduser -D -h /home/dev dev && \
    echo "dev ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers

ENV TZ=Asia/Jakarta
WORKDIR /home/dev

EXPOSE 9998

CMD sh -c 'exec ttyd -p 9998 -c "dev:${DEV_PASS:-devpass}" bash'
