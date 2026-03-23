# oc
FROM brew.registry.redhat.io/rh-osbs/openshift-ose-cli-rhel9:v4.22 AS ose-cli

FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_9_golang_1.25 AS builder

COPY . /workspace
USER root
RUN chown -R 1001:0 /workspace
USER 1001

# mustgather
WORKDIR /workspace/
RUN CGO_ENABLED=1 GOOS=linux go build -a -mod=mod -tags strictfipsruntime -o ./gather cmd/main.go

# kopia
WORKDIR /workspace/kopia/
RUN CGO_ENABLED=1 GOOS=linux go build -a -mod=mod -tags strictfipsruntime -o ./kopia

#######################################################################
#######################################################################
#                                                                     #
#      W     W    AA     RRRR     N   N    III    N   N     GGG       #
#      W     W   A  A    R   R    NN  N     I     NN  N    G          #
#      W  W  W   AAAA    RRRR     N N N     I     N N N    G  GG      #
#       W W W    A  A    R R      N  NN     I     N  NN    G   G      #
#        W W     A  A    R  RR    N   N    III    N   N     GGG       #
#                                                                     #
#  Any changes to the `velero` and `restic` sections below must also  #
#  be reconciled in oadp-velero/Dockerfile.in for consistency.        #
#######################################################################
# BEGIN                                                               #
#######################################################################

# velero
WORKDIR /workspace/velero/
ENV GOEXPERIMENT strictfipsruntime
RUN CGO_ENABLED=1 GOOS=linux go build -a -mod=mod -ldflags '-X github.com/vmware-tanzu/velero/pkg/buildinfo.Version=v1.16.1-OADP' -tags strictfipsruntime -o ./bin/velero ./cmd/velero

# restic
WORKDIR /workspace/restic/
ENV GOEXPERIMENT strictfipsruntime
RUN CGO_ENABLED=1 GOOS=linux go build -a -mod=mod -tags strictfipsruntime -o ./bin/restic ./cmd/restic

#######################################################################
# END                                                                 #
#######################################################################

ENV INSTALLATION_NAMESPACE openshift-adp

FROM registry.redhat.io/ubi9/ubi:latest
RUN dnf -y install openssl rsync tar gzip && dnf -y reinstall tzdata && dnf -y clean all

COPY --from=builder /workspace/velero/bin/velero /usr/bin/velero
COPY --from=builder /workspace/restic/bin/restic /usr/bin/restic
COPY --from=ose-cli /usr/bin/oc /usr/bin/oc
COPY --from=builder /workspace/kopia/kopia /usr/bin/kopia
COPY --from=builder /workspace/gather /usr/bin/gather
COPY --from=builder /workspace/deprecated/gather_* /usr/bin/
COPY --from=builder /workspace/LICENSE /licenses/

ENTRYPOINT /usr/bin/gather

LABEL description="OpenShift API for Data Protection data gathering image"
LABEL io.k8s.description="OpenShift API for Data Protection data gathering image"
LABEL io.k8s.display-name="OpenShift API for Data Protection - mustgather"
LABEL io.openshift.tags="data,images"
LABEL summary="OpenShift API for Data Protection data gathering image"
