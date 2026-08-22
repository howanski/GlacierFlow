FROM alpine:3.22.4

ARG GF_USER_ID=1000
ARG GF_PI_EXTRA_PACKAGES=""

RUN adduser -D -u ${GF_USER_ID} pi_dev

# BASIC REQUIREMENTS
RUN apk add bash curl ttyd tmux git npm vim

# USER-DEFINED PACKAGES (space-separated, from .env via compose build-arg)
RUN if [ -n "$GF_PI_EXTRA_PACKAGES" ]; then \
        apk add $GF_PI_EXTRA_PACKAGES; \
    fi

USER pi_dev

CMD ["/pi_dev/entrypoint"]
