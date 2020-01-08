FROM golang:1.12.1-stretch

# kubebuilder
ARG version=1.0.8
ARG arch=amd64
RUN curl -L -O "https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${version}/kubebuilder_${version}_linux_${arch}.tar.gz" && \
    tar -zxvf kubebuilder_${version}_linux_${arch}.tar.gz && \
    mv kubebuilder_${version}_linux_${arch} kubebuilder && \
    mv kubebuilder /usr/local/
ENV PATH="/usr/local/kubebuilder/bin:${PATH}"
