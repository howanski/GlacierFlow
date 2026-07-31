FROM alpine:3.22.4

ARG GF_USER_ID=1000
RUN adduser -D -u ${GF_USER_ID} pi_dev

RUN apk add bash curl ttyd tmux libstdc++ gcompat git npm vim rust cargo php go openjdk17 gradle android-tools composer

RUN curl -fsSL https://pi.dev/install.sh | sh

RUN pi install git:github.com/badlogic/pi-skills

USER pi_dev

CMD ["/pi_dev/entrypoint"]
