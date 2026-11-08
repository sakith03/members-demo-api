FROM golang:1.27.3-alpine AS build
WORKDIR /src
COPY go.mod main.go ./
COPY static ./static
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/members-frontend .

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/members-frontend /members-frontend
USER 65532:65532
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/members-frontend"]
