module gitlab.com/kpouget_psap/specfem-client

go 1.14

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => github.com/openshift/kubernetes-client-go v0.0.0-20200507115529-5e2a2d83bced
)

require (
	github.com/fsouza/go-dockerclient v1.6.5 // indirect
	github.com/gonum/blas v0.0.0-20181208220705-f22b278b28ac // indirect
	github.com/gonum/floats v0.0.0-20181209220543-c233463c7e82 // indirect
	github.com/gonum/graph v0.0.0-20190426092945-678096d81a4b // indirect
	github.com/gonum/internal v0.0.0-20181124074243-f884aa714029 // indirect
	github.com/gonum/lapack v0.0.0-20181123203213-e4cdc5a0bff9 // indirect
	github.com/gonum/matrix v0.0.0-20181209220409-c518dec07be9 // indirect
	github.com/imdario/mergo v0.3.10 // indirect
	github.com/openshift/api v0.0.0-20200710154525-af4dd20aed23
	github.com/openshift/client-go v0.0.0-20200521150516-05eb9880269c
	github.com/openshift/oc v4.2.0-alpha.0+incompatible
	github.com/openshift/origin v0.0.0-20160503220234-8f127d736703
	gitlab.com/kpouget_psap/specfem-operator v0.0.0-20200721133333-0c653d3eec3d
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	k8s.io/api v0.18.5
	k8s.io/apimachinery v0.18.5
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/utils v0.0.0-20200716102541-988ee3149bb2 // indirect
	sigs.k8s.io/structured-merge-diff/v2 v2.0.1 // indirect
)
