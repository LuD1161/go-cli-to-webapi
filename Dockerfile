FROM golang:1.16 as builder

RUN mkdir /app
ADD . /app
WORKDIR /app

RUN CGO_ENABLED=0 GOOS=linux go build -o app

FROM alpine:latest as production
COPY --from=builder /app .
# Setting up PATH for the tools
ENV PATH "$PATH:/tools"
# Downloading libc to get go binaries to execute. https://stackoverflow.com/a/50861580
# Other solution is to create a totally independent go binary. 
# But that's probably not implemented for all the platforms, as mentioned in the SO answer above
RUN apk add --no-cache libc6-compat && mkdir tools

# Setup your tools here

# # Setting up nuclei
RUN wget https://github.com/projectdiscovery/nuclei/releases/download/v2.3.8/nuclei_2.3.8_linux_amd64.tar.gz && tar -xvzf nuclei_2.3.8_linux_amd64.tar.gz && chmod +x nuclei && mv nuclei ./tools/ && rm nuclei_2.3.8_linux_amd64.tar.gz

# Run the binary
CMD [ "./app" ]