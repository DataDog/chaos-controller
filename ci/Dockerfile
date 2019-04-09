FROM golang:1.12.1-stretch

# kubebuilder
ARG version=1.0.8
ARG arch=amd64
RUN curl -L -O "https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${version}/kubebuilder_${version}_linux_${arch}.tar.gz" && \
    tar -zxvf kubebuilder_${version}_linux_${arch}.tar.gz && \
    mv kubebuilder_${version}_linux_${arch} kubebuilder && \
    mv kubebuilder /usr/local/
ENV PATH="/usr/local/kubebuilder/bin:${PATH}"

# docker, aws and gcp
RUN \
  apt update && apt install -y apt-transport-https && \
  echo "deb http://packages.cloud.google.com/apt cloud-sdk-stretch main" > \
    /etc/apt/sources.list.d/google-cloud-sdk.list && \
  echo "deb [arch=amd64] https://download.docker.com/linux/debian stretch stable" > \
    /etc/apt/sources.list.d/docker.list && \
  curl -fsL https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add - && \
  curl -fsL https://download.docker.com/linux/debian/gpg | apt-key add - && \
  apt update && apt install -y awscli google-cloud-sdk docker-ce-cli python-pip && \
  pip install awscli
