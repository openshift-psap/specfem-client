package main

import (
	"fmt"
	"log"

	"k8s.io/apimachinery/pkg/runtime/schema"

	specfemv1 "github.com/openshift-psap/specfem-client-api/pkg/apis/specfem/v1alpha1"
)

var imagestreamtagResource         = schema.GroupVersionResource{Version: "v1", Resource: "imagestreamtags", Group: "image.openshift.io"}
var podResource         = schema.GroupVersionResource{Version: "v1", Resource: "pods"}
var pvcResource         = schema.GroupVersionResource{Version: "v1", Resource: "persistentvolumeclaims"}
var cmResource          = schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}
var svcResource         = schema.GroupVersionResource{Version: "v1", Resource: "services"}
var jobResource         = schema.GroupVersionResource{Version: "v1", Resource: "jobs",         Group: "batch"}
var buildconfigResource = schema.GroupVersionResource{Version: "v1", Resource: "buildconfigs", Group: "build.openshift.io"}
var imagestreamResource = schema.GroupVersionResource{Version: "v1", Resource: "imagestreams", Group: "image.openshift.io"}
var routeResource       = schema.GroupVersionResource{Version: "v1", Resource: "routes",       Group: "route.openshift.io"}
var secretResource       = schema.GroupVersionResource{Version: "v1", Resource: "secrets"}

var NAMESPACE = ""

var USE_UBI_BASE_IMAGE = true

type TemplateCfg struct {
	ConfigMaps struct {
		HelperFile struct {
			ConfigMapName string
			ManifestName string
		}
	}
	SecretNames struct {
		DockerCfgPush string
	}
	MesherSolver struct {
		Stage string
		Image string
		Nreplicas int
	}
}

func yamlImageStream() (string, YamlResourceTmpl) {
	return "00_imagestream.yaml", NoTemplateCfg
}

func yamlBaseImageBuildConfig() (string, YamlResourceTmpl) {
	return "01_buildconfig_base.yaml", NoTemplateCfg
}

func yamlMesherImageBuildConfig() (string, YamlResourceTmpl) {
	return "02_buildconfig_mesher.yaml", NoTemplateCfg
}

func yamlPVC() (string, YamlResourceTmpl) {
	return "03_pvc.yaml", NoTemplateCfg
}

func yamlMesherScriptCM() (string, YamlResourceTmpl) {
	return "99_configmap_helper-files.yaml", func(app *specfemv1.SpecfemApp) *TemplateCfg {
		cfg := &TemplateCfg{}
		cfg.ConfigMaps.HelperFile.ManifestName = "run_mesher.sh"
		return cfg
	}
}

func newMesher2SolverHelperBuildConfig() (string, YamlResourceTmpl) {
	return "05a_buildconfig_mesher2solver-helper.yaml", NoTemplateCfg
}

func yamlBuildahMesher2SolverScriptCM() (string, YamlResourceTmpl) {
	return "99_configmap_helper-files.yaml", func(app *specfemv1.SpecfemApp) *TemplateCfg {
		cfg := &TemplateCfg{}
		cfg.ConfigMaps.HelperFile.ManifestName = "run_mesher2solver.sh"
		return cfg
	}
}

func yamlTunedLoadFuseModule() (string, YamlResourceTmpl) {
	return "05b_tuned_fuse-module.yaml", NoTemplateCfg
}

func yamlBuildahBuildSolverImagePod() (string, YamlResourceTmpl) {
	pushsecretName, err:= getPushSecretName()
	if err != nil {
		log.Fatalf("FATAL: failed to get push secret: %+v", err)
	}

	log.Printf("Using push secret '%s'", pushsecretName)

	return "05c_job_mesher2solver-builder.yaml", func(app *specfemv1.SpecfemApp) *TemplateCfg {
		cfg := &TemplateCfg{}
		cfg.SecretNames.DockerCfgPush = pushsecretName
		return cfg
	}
}

func yamlSolverScriptCM() (string, YamlResourceTmpl) {
	return "99_configmap_helper-files.yaml", func(app *specfemv1.SpecfemApp) *TemplateCfg {
		cfg := &TemplateCfg{}
		cfg.ConfigMaps.HelperFile.ManifestName = "run_solver.sh"
		return cfg
	}
}

func yamlRunSeqMesherSolverJob(stage string) YamlResourceSpec {
	return func() (string, YamlResourceTmpl) {
		return "99_job_meshersolver-seq.yaml", func(app *specfemv1.SpecfemApp) *TemplateCfg {
			cfg := &TemplateCfg{}
			cfg.MesherSolver.Stage = stage
			return cfg
		}
	}
}

func yamlRunMpiMesherSolverJob(stage string) YamlResourceSpec {
	return func() (string, YamlResourceTmpl) {
		return "99_mpijob_meshersolver.yaml", func(app *specfemv1.SpecfemApp) *TemplateCfg {
			cfg := &TemplateCfg{}
			cfg.MesherSolver.Stage = stage
			cfg.MesherSolver.Image += fmt.Sprintf("%s-%dproc-%dnex",
				stage, app.Spec.Exec.Nproc, app.Spec.Specfem.Nex)

			cfg.MesherSolver.Nreplicas = int(app.Spec.Exec.Nproc/app.Spec.Exec.SlotsPerWorker)

			return cfg
		}
	}
}

func yamlSaveSolverOutputJob() (string, YamlResourceTmpl) {
	return "06_job_save-solver-output.yaml", NoTemplateCfg
}
