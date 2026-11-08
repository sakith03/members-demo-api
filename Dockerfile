FROM golang:1.27.3-alpine AS build
WORKDIR /src
COPY go.mod main.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/members-api .

FROM scratch
COPY --from=build /out/members-api /members-api
USER 65532:65532
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/members-api"]
