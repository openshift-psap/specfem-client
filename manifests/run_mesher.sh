#! /bin/bash

set -e

mkdir /mnt/shared/DATABASES_MPI /mnt/shared/OUTPUT_FILES -p

cd app 

./bin/xmeshfem3D

if [[ -z "$OMPI_COMM_WORLD_RANK" || $OMPI_COMM_WORLD_RANK -eq 0 ]]; then
  cat OUTPUT_FILES/output_mesher.txt | grep "buffer creation in seconds"
  env > /mnt/shared/env.mesher
fi
