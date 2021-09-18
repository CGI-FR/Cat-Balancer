FROM gcr.io/distroless/base
ARG BIN
COPY /bin/cb /cb
CMD ["/cb"]
