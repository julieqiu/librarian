# Copyright 2024 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:1.24 AS build

WORKDIR /src

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY cmd cmd
COPY internal internal
RUN CGO_ENABLED=0 GOOS=linux go build ./cmd/librarian

# Using docker:dind so we can run docker from the CLI,
# while in Docker. Note that for this to work, *this*
# docker image should be run with
#  -v /var/run/docker.sock:/var/run/docker.sock
FROM golang:1.24
WORKDIR /app

# From https://docs.docker.com/engine/install/debian/
# Add Docker's official GPG key
RUN apt update
RUN apt-get install -y unzip && rm -rf /var/lib/apt/lists/*
RUN apt install -y ca-certificates curl
RUN install -m 0755 -d /etc/apt/keyrings
RUN curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc
RUN chmod a+r /etc/apt/keyrings/docker.asc

# Add the repository to Apt sources:
RUN echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian \
    $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
      tee /etc/apt/sources.list.d/docker.list > /dev/null
RUN apt update

# Install Docker
RUN apt-get -y install docker-ce

COPY --from=build /src/librarian .
ENTRYPOINT ["/app/librarian"]
