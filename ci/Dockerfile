FROM golang:1.16.0-buster

# kubebuilder
ARG version=2.3.1
ARG arch=amd64
RUN curl -L -O "https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${version}/kubebuilder_${version}_linux_${arch}.tar.gz" && \
    tar -zxvf kubebuilder_${version}_linux_${arch}.tar.gz && \
    mv kubebuilder_${version}_linux_${arch} kubebuilder && \
    mv kubebuilder /usr/local/

# golangci-lint
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.23.1

ENV PATH="/usr/local/kubebuilder/bin:${PATH}"
