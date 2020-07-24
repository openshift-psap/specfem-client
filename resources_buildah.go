package main

import (
	"log"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	
	buildv1 "github.com/openshift/api/build/v1"

	specfemv1 "gitlab.com/kpouget_psap/specfem-operator/pkg/apis/specfem/v1"
	"k8s.io/apimachinery/pkg/runtime"
	//"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/runtime/schema"

    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
    "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
)

/*
import tunedv1 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/tuned/v1"
|
V
go: finding module for package github.com/openshift/cluster-node-tuning-operator/pkg/apis/tuned/v1
go: found github.com/openshift/cluster-node-tuning-operator/pkg/apis/tuned/v1 in github.com/openshift/cluster-node-tuning-operator v0.0.0-20200716150318-9bfd3ea95d29
go: github.com/openshift/cluster-node-tuning-operator@v0.0.0-20200716150318-9bfd3ea95d29 requires
	github.com/openshift/api@v3.9.1-0.20191111211345-a27ff30ebf09+incompatible: invalid pseudo-version: preceding tag (v3.9.0) not found
*/

var tunedResource         = schema.GroupVersionResource{Version: "v1", Resource: "tuneds", Group: "tuned.openshift.io"}

func newTunedLoadFuseModule(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {

	objName := "specfem-fuse-for-buildah"
	
const dsManifest = `
# if oc get nodes -l buildah.specfem.build |& grep -q "No resources found"; then
#    node_to_tag=$(oc get nodes -l node-role.kubernetes.io/worker | head -2 | tail -1 | cut -d" " -f1)
#    echo "Tagging node '$node_to_tag to run fuse+buildah "
#    oc label node $node_to_tag buildah.specfem.build=
# fi

apiVersion: tuned.openshift.io/v1
kind: Tuned
metadata:
  name: %v
spec:
  profile:
  - data: |
      [main]
      summary=An OpenShift profile to load 'fuse' module
      include=openshift-node
      [modules]
      fuse=+r
    name: openshift-fuse
  recommend:
  - match:
    - label: buildah.specfem.build
    profile: "openshift-fuse"
    priority: 5
`
	obj := &unstructured.Unstructured{}

    // decode YAML into unstructured.Unstructured
    dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
    _, _, err := dec.Decode([]byte(fmt.Sprintf(dsManifest, objName)), nil, obj)
		
	if err != nil {
		return tunedResource, "", nil
	}

	return tunedResource, objName, obj
}

func newAfterMeshHelperBuildConfig(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	objName := "specfem-after-mesh-helper"
	
	return buildconfigResource, objName, &buildv1.BuildConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objName,
			Namespace: NAMESPACE,
			Labels:    map[string]string{
				"app": "specfem",
			},
		},
		Spec: buildv1.BuildConfigSpec{
			CommonSpec: buildv1.CommonSpec{
				Strategy: buildv1.BuildStrategy{
					Type: buildv1.DockerBuildStrategyType,
					DockerStrategy: &buildv1.DockerBuildStrategy{
						From: &corev1.ObjectReference{
							Kind: "DockerImage",
							Name: "docker.io/centos:7",
						},
					},
				},
				Source: buildv1.BuildSource{
					Type: buildv1.BuildSourceGit,
					Git: &buildv1.GitBuildSource{
						URI: "https://gitlab.com/kpouget_psap/specfem-on-openshift.git",
						Ref: "02_specfem-solver-container_buildah",
					},
				},
				Output: buildv1.BuildOutput{
					To: &corev1.ObjectReference{
						Kind: "ImageStreamTag",
						Name: "specfem:mesher2solver_helper",
					},
				},
			},
			Triggers: []buildv1.BuildTriggerPolicy{
				buildv1.BuildTriggerPolicy{
					Type: buildv1.ConfigChangeBuildTriggerType,
				},
			},
		},
	}
}

func newBuildahBuildSolverImagePod(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	f32 := func(s int32) *int32 {
        return &s
    }
	boolPtr := func(v bool) *bool {
		t := true
		f := false
		if v {
			return &t
		} else {
			return &f
		}
	}
	objName := "buildah-build-solver-image-pod"

	return podResource, objName, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objName,
			Namespace: NAMESPACE,
			Labels:    map[string]string{
				"app": "specfem",
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			NodeSelector: map[string]string{
				"buildah.specfem.build": "",
			},
			Containers: []corev1.Container{
				corev1.Container{
					
					Name:  objName,
					Image: "image-registry.openshift-image-registry.svc:5000/"+NAMESPACE+"/specfem:mesher2solver_helper",
					Command: []string{
						"/bin/sh", "/mnt/helper/run.sh",
					},
					SecurityContext: &corev1.SecurityContext{
						Privileged: boolPtr(true),
					},
					VolumeMounts: []corev1.VolumeMount{
						corev1.VolumeMount{
							Name: "shared-volume",
							MountPath: "/mnt/shared/",
						},
						corev1.VolumeMount{
							Name: "bash-run-buildah-helper",
							MountPath: "/mnt/helper/run.sh",
							ReadOnly: true,
							SubPath: "run.sh",
						},
						corev1.VolumeMount{
							Name: "builder-dockercfg-push",
							MountPath: "/var/run/secrets/openshift.io/push",
							ReadOnly: true,
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				corev1.Volume{
					Name: "shared-volume",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "specfem",
						},
					},
				},
				corev1.Volume{
					Name: "bash-run-buildah-helper",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "bash-run-buildah-helper",
							},
							DefaultMode: f32(0777),
						},
					},
				},
				corev1.Volume{
					Name: "builder-dockercfg-push",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "builder-dockercfg-bbskl", // TODO: find dynamically
							DefaultMode: f32(384),
						},
					},
				},
			},						
		},
	}
}

func newBuildahAfterMeshScriptCM(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	objName := "bash-run-buildah-helper"

	return cmResource, objName, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objName,
			Namespace: NAMESPACE,
			Labels:    map[string]string{
				"app": "specfem",
			},
		},
		Data: map[string]string{
			"run.sh": `
set -xe

test -c /dev/fuse # mknod /dev/fuse c 10 229

cp /var/run/secrets/openshift.io/push/.dockercfg /tmp
(echo "{ \"auths\": " ; cat /var/run/secrets/openshift.io/push/.dockercfg ; echo "}") > /tmp/.dockercfg

AUTH="--tls-verify=false --authfile /tmp/.dockercfg"

IMG_STREAM=image-registry.openshift-image-registry.svc:5000/`+NAMESPACE+`/specfem
cont=$(buildah $AUTH from $IMG_STREAM:mesher)

buildah run $cont bash -c 'echo "$(date) | Using BUILDAH --volume to build the solver  ..." >> /app/oc.build.log'

buildah run --volume /mnt/shared:/mnt/shared:rw,z $cont bash -c '\
        echo "$(date) | Building the solver ..." >> /app/oc.build.log && \
        cd app &&  \
        mkdir obj && \
        make spec && \
        rm obj/ -rf && \
        chmod 777 /app -R'

cont_img=$(buildah commit $AUTH $cont)
buildah push $AUTH $cont_img $IMG_STREAM:solver
`,
		},
	}
}

func CreateSolverImage_buildah(app *specfemv1.SpecfemApp) error {
	if err:= CreateAndWaitBuildConfig(app, newAfterMeshHelperBuildConfig, "buildah"); err != nil {
		return err
	}
	
	for _, creatorFunction := range[]ResourceCreator{
		newBuildahAfterMeshScriptCM, newTunedLoadFuseModule} {	
		if _, err := CreateResource(app, creatorFunction, "buildah"); err != nil {
			return err
		}
	}

	if !delete_mode {
		if err := CheckImageTag(app, "specfem:solver", "mesher"); err == nil {
			log.Println("Found solver image, don't recreate it.")
			return nil
		}
		
		if err := WaitForTunedProfile(app, "openshift-fuse", "ip-10-0-145-12.us-east-2.compute.internal"); err != nil {
			return err
		}
	}
	
	podName, err := CreateResource(app, newBuildahBuildSolverImagePod, "mesher")
	if err != nil {
		return err
	}

	if podName != "" {
		if err := WaitWithPodLogs("", podName, "", nil); err != nil {
			return err
		}
	}

	if err := CheckImageTag(app, "specfem:solver", "mesher"); err != nil {
		return err
	}
	
	return nil
}
