package main

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	batchv1 "k8s.io/api/batch/v1"
	buildv1 "github.com/openshift/api/build/v1"
	routev1 "github.com/openshift/api/route/v1"

	specfemv1 "gitlab.com/kpouget_psap/specfem-operator/pkg/apis/specfem/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/runtime/schema"

)

func newPyServerOneFileCM(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	objName := "py-serve-one-file"
	return cmResource, objName, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objName,
			Namespace: NAMESPACE,
			Labels:    map[string]string{
				"app": "specfem",
			},
		},
		Data: map[string]string{
			"serve-one-file.py": `
import sys, os
from http.server import HTTPServer, BaseHTTPRequestHandler

def main():
    PORT = 8000
    ADDR = "0.0.0.0"
    try: file_to_serve = sys.argv[1]
    except IndexError:
        print("Please pass a file to serve in argument ...")
        exit(1)

    if not os.path.exists(file_to_serve):
        print("File not found: '{}' (in {}) ...".format(file_to_serve, os.getcwd()))
        exit(2)

    class SimpleHTTPRequestHandler(BaseHTTPRequestHandler):

        def do_GET(self):
            self.send_response(200)
            self.send_header('Content-type','text/plain')
            self.end_headers()
            with open(file_to_serve, "rb") as shared_f:
                for b_line in shared_f.readlines():
                    self.wfile.write(b_line)

    httpd = HTTPServer((ADDR, PORT), SimpleHTTPRequestHandler)

    print("Listening on {}:{} ...".format(ADDR, PORT), flush=True)

    httpd.handle_request()
    print("Done!".format(ADDR, PORT))

if __name__ == "__main__":
    main()
`,
		},
	}
}

func newShareMesherValuesJob(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	f32 := func(s int32) *int32 {
        return &s
    }
	f64 := func(s int64) *int64 {
        return &s
    }

	objName := "share-mesher-values"
	return jobResource, objName, &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objName,
			Namespace: NAMESPACE,
			Labels:    map[string]string{
				"app": "specfem",
			},
		},
		Spec: batchv1.JobSpec{
			Parallelism: f32(1),
			Completions: f32(1),
			ActiveDeadlineSeconds: f64(300),
			BackoffLimit: f32(1),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "share-mesher-values",
					Namespace: NAMESPACE,
					Labels:    map[string]string{
						"app": "specfem",
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						corev1.Container{
							Name: "specfem-share-mesher",
							ImagePullPolicy: corev1.PullAlways,
							Image: fmt.Sprintf("image-registry.openshift-image-registry.svc:5000/%v/specfem:mesher", NAMESPACE),
							Ports: []corev1.ContainerPort{
								corev1.ContainerPort{
									Name: "container-port",
									ContainerPort: 8000,
								},
							},
							Command: []string{
								"bash", "-c",
								`echo "Sharing mesher outputs ..."; test -e /mnt/shared/mesher.tgz && exec python3 /mnt/helper/serve-one-file.py /mnt/shared/mesher.tgz`,
							},
							VolumeMounts: []corev1.VolumeMount{
								corev1.VolumeMount{
									Name: "shared-volume",
									MountPath: "/mnt/shared/",
								},
								corev1.VolumeMount{
									Name: "py-helper-volume",
									MountPath: "/mnt/helper/serve-one-file.py",
									ReadOnly: true,
									SubPath: "serve-one-file.py",
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
							Name: "py-helper-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "py-serve-one-file",
									},
									DefaultMode: f32(0700),
								},
							},
						},
					},
				},
			},
		},
	}
}

func newValuesFromMesherService(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	objName :=  "specfem-share-mesh"

	return svcResource, objName, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objName,
			Namespace: NAMESPACE,
			Labels:    map[string]string{
				"app": "specfem",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"job-name": "share-mesher-values",
			},
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Port: 8000,
					TargetPort: intstr.FromInt(8000),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
}

func newValuesFromMesherRoute(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	f32 := func(s int32) *int32 {
        return &s
    }

	objName := "specfem-share-mesh"

	return routeResource, objName, &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objName,
			Namespace: NAMESPACE,
			Labels:    map[string]string{
				"app": "specfem",
			},
		},
		Spec: routev1.RouteSpec{
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromInt(8000),
			},
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: "specfem-share-mesh",
				Weight: f32(100),
			},
			WildcardPolicy: routev1.WildcardPolicyNone,
		},
	}
}

func newCurlSolverImageBuildConfig(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	objName := "specfem-solver-image-curl"

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
						ForcePull: true,
						From: &corev1.ObjectReference{
							Kind: "ImageStreamTag",
							Name: "specfem:mesher",
						},
					},
				},
				Source: buildv1.BuildSource{
					Type: buildv1.BuildSourceGit,
					Git: &buildv1.GitBuildSource{
						URI: BASE_GIT_REPO,
						Ref: "02_specfem-solver-container_curl",
					},
				},
				Output: buildv1.BuildOutput{
					To: &corev1.ObjectReference{
						Kind: "ImageStreamTag",
						Name: "specfem:solver",
					},
				},
			},
			Triggers: []buildv1.BuildTriggerPolicy{
				buildv1.BuildTriggerPolicy{
					Type: buildv1.ImageChangeBuildTriggerType,
				},
			},
		},
	}
}

func CreateSolverImage_port(app *specfemv1.SpecfemApp) error {
	for _, creatorFunction := range[]ResourceCreator{
		newPyServerOneFileCM, newValuesFromMesherRoute, newValuesFromMesherService} {	
		if _, err := CreateResource(app, creatorFunction, "port"); err != nil {
			return err
		}
	}

	jobName, err := CreateResource(app, newShareMesherValuesJob, "mesher")
	if err != nil {
		return err
	}

	if jobName != "" {
		if err := WaitWithJobLogs(jobName, "Listening on", nil); err != nil {
			return err
		}
	}

	if err:= CreateAndWaitBuildConfig(app, newCurlSolverImageBuildConfig, "mesher"); err != nil {
		return err
	}

	if err := CheckImageTag(app, "specfem:solver", "config"); err != nil {
		return err
	}

	return nil
}
