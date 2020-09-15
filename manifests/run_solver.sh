#! /bin/bash

set -e

if [[ "$SPECFEM_RELY_ON_SHARED_FS" != "true" ]]; then
    if [[ "$OMPI_COMM_WORLD_NODE_RANK" -eq 0 ]]; then
        rm -f /app/{DATABASES_MPI,OUTPUT_FILES} # remove the symlinks
        cp /mnt/shared/{DATABASES_MPI,OUTPUT_FILES} /app/ -r
    fi
fi

cd /app && ./bin/xspecfem3D

if [[ -z "$OMPI_COMM_WORLD_RANK" || $OMPI_COMM_WORLD_RANK -eq 0 ]]; then
  cp /app/oc.build.log /mnt/shared/
  env > /mnt/shared/env.solver
  cat /app/OUTPUT_FILES/output_solver.txt | grep "Total elapsed time in hh:mm:ss"
fi

[[ "$SPECFEM_RELY_ON_SHARED_FS" == "true" ]] && exit 0

if [[ -z "$OMPI_COMM_WORLD_RANK" || $OMPI_COMM_WORLD_RANK -eq 0 ]]; then
    cp /app/OUTPUT_FILES/* /mnt/shared/OUTPUT_FILES/
fi

echo Solver done $OMPI_COMM_WORLD_RANK
