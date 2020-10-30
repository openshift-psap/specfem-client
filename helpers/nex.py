#! /usr/bin/python3 

# Helper script to know the possible combination of NEX/NPROC/Machines

NEX_MAX=256
M = lambda m : [m * i for i in range(1, NEX_MAX+1)]
T = lambda n, p: n in M(16) and n in M(8*p)
U = lambda u : [i for i in range(1, NEX_MAX+1) if T(i, u)]

ALL = U(1)

NPROC = 8
N_MACHINES = 32
N_MACHINES_CURRENT = 16

NPROC_NEX = {}

for nproc in range(NPROC+1):
    NPROC_NEX[nproc] = U(nproc)


for nex in ALL:
    procs = [nproc*nproc for nproc, nproc_nex in NPROC_NEX.items() if nex in nproc_nex]
    if len(procs) <= 2: continue
    print(f"# {nex} NEX")
    for nproc in procs[1:]:
        if nproc > 2*N_MACHINES: continue
        if nproc % 2 == 0:
            nproc_2 = int(nproc/2)
            if nproc_2 not in procs or nproc > N_MACHINES:
                if nproc_2 > N_MACHINES_CURRENT: print("#", end="")
                print(f"- processes={nproc}, mpi-slots=2, nex={nex} # {nproc_2} machines")
            
        if nproc > N_MACHINES: continue
        if nproc > N_MACHINES_CURRENT: print("#", end="")
        print(f"- processes={nproc}, mpi-slots=1, nex={nex} # {nproc} machines")
    print()
print()

for nex in ALL:
    procs = [nproc*nproc for nproc, nproc_nex in NPROC_NEX.items() if nex in nproc_nex]
    if len(procs) <= 2: continue
    print(f"# {nex:>3d} NEX=", end="")
    for nproc in procs[1:]:
        if nproc > 4*N_MACHINES: continue
        if nproc % 4 == 0:
            nproc_4 = int(nproc/4)
            if nproc_4 not in procs or nproc > N_MACHINES:
                print(f" {nproc_4:>4d}(x4)", end="")
        if nproc % 2 == 0:
            nproc_2 = int(nproc/2)
            if nproc_2 not in procs or nproc > N_MACHINES:
                print(f" {nproc_2:>4d}(x2)", end="")

                
        if nproc > N_MACHINES: continue            
        print(f" {nproc:>4d}    ", end="")
    print()
