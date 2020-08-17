package main

var dockerfile_base_container_ubi = `
FROM registry.access.redhat.com/ubi8/ubi

RUN dnf -y install sudo pkg-config vim  make gdb \
    curl git openssh-clients \
    gcc-gfortran gcc-c++ \
    openmpi-devel openmpi \
 && ln -s /usr/lib64/openmpi/bin/orted /usr/bin/orted

ENV PATH="${PATH}:/usr/lib64/openmpi/bin/"

RUN git clone $SPECFEM_GIT_REPO -b $SPECFEM_GIT_BRANCH --depth 1 /app \
 && echo "$(date) | Cloned $SPECFEM_GIT_REPO branch $SPECFEM_GIT_BRANCH (git rev-parse --short HEAD)" >> /app/oc.build.log \
 && rm -rf /app/.git
`

var dockerfile_base_container_ubuntu = `
FROM docker.io/ubuntu:eoan

# Install Deps
RUN apt-get update && \
    apt-get install -y \
            sudo \
            pkg-config vim \
            g++ make \
            gdb strace \
            curl git \
            gfortran libgomp1 openmpi-bin libopenmpi-dev \
            ssh # solves the issue with mpirun failing to launch anything

ENV PATH="${PATH}:/usr/lib64/openmpi/bin/"

RUN git clone $SPECFEM_GIT_REPO -b $SPECFEM_GIT_BRANCH --depth 1 /app \
 && echo "$(date) | Cloned $SPECFEM_GIT_REPO branch $SPECFEM_GIT_BRANCH (git rev-parse --short HEAD)" >> /app/oc.build.log \
 && rm -rf /app/.git
`
var dockerfile_mesher_container = `
FROM specfem:base

RUN cd /app \
 && ./configure --enable-openmp FLAGS_CHECK=-Wno-error

RUN echo "$(date) | Configuring Specfem DATA/Par_file from env ..." >> /app/oc.build.log \
 && echo "$(date) |     OMP_NUM_THREADS=$OMP_NUM_THREADS" >> /app/oc.build.log \
 && echo "$(date) |     SPECFEM_NEX=$SPECFEM_NEX" >> /app/oc.build.log \
 && echo "$(date) |     SPECFEM_NPROC=$SPECFEM_NPROC" >> /app/oc.build.log \
 && sed -i -e "s/NEX_XI[ ]*= .*/NEX_XI = $SPECFEM_NEX/" /app/DATA/Par_file   && grep "NEX_XI = $SPECFEM_NEX" /app/DATA/Par_file \
 && sed -i -e "s/NEX_ETA[ ]*= .*/NEX_ETA = $SPECFEM_NEX/" /app/DATA/Par_file && grep "NEX_ETA = $SPECFEM_NEX" /app/DATA/Par_file \
 && sed -i -e "s/NPROC_XI[ ]*= .*/NPROC_XI = $SPECFEM_NPROC/" /app/DATA/Par_file   && grep "NPROC_XI = $SPECFEM_NPROC" /app/DATA/Par_file \
 && sed -i -e "s/NPROC_ETA[ ]*= .*/NPROC_ETA = $SPECFEM_NPROC/" /app/DATA/Par_file && grep "NPROC_ETA = $SPECFEM_NPROC" /app/DATA/Par_file

RUN echo "$(date) | Building the mesher ..." >> /app/oc.build.log \
 && cd /app \
 && make mesh \
 && rm .git obj/ OUTPUT_FILES DATABASES_MPI -rf \
 && ln -s /mnt/shared/OUTPUT_FILES \
 && ln -s /mnt/shared/DATABASES_MPI \
 && chmod 777 /app -R
`
