#! /bin/bash

set -xe

cp /var/run/secrets/openshift.io/push/.dockercfg /tmp
(echo "{ \"auths\": " ; cat /var/run/secrets/openshift.io/push/.dockercfg ; echo "}") > /tmp/.dockercfg

AUTH="--tls-verify=false --authfile /tmp/.dockercfg"

IMG_SUFFIX="${SPECFEM_MPI_NPROC}proc-${SPECFEM_NEX}nex"

IMG_STREAM=image-registry.openshift-image-registry.svc:5000/{{ .App.ObjectMeta.Namespace }}/specfem
cont=$(buildah $AUTH from $IMG_STREAM:mesher-$IMG_SUFFIX)

buildah run $cont bash -c 'echo "$(date) | Using BUILDAH --volume to build the solver  ..." >> /app/oc.build.log'

CACHE_NAME=cache/${SPECFEM_MPI_NPROC}proc/${SPECFEM_NEX}nex/

buildah run --volume /mnt/shared/$CACHE_NAME:/mnt/shared/:rw,z $cont bash -c '\
        echo "$(date) | Building the solver ..." >> /app/oc.build.log && \
        cd app &&  \
        mkdir obj && \
        ln -s /mnt/shared/DATABASES_MPI && \
        ln -s /mnt/shared/OUTPUT_FILES && \
        make spec && \
        rm DATABASES_MPI OUTPUT_FILES && \
        rm obj/ -rf && \
        chmod 777 /app -R'

cont_img=$(buildah commit $AUTH $cont)
buildah push $AUTH $cont_img $IMG_STREAM:solver-$IMG_SUFFIX

