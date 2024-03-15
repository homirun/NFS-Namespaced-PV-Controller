package controller

import (
	"context"
	"os"
	"os/exec"
	"time"

	namespacedpvv1 "github.com/homirun/namespaced-pv-controller/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var _ = Describe("namespaced pv controller e2e test", func() {
	ctx := context.Background()
	var c client.Client
	BeforeEach(func() {
		c = prepare(ctx)
		pvs := &corev1.PersistentVolumeList{}
		err := c.List(ctx, pvs)
		Expect(err).NotTo(HaveOccurred())
		for _, pv := range pvs.Items {
			err = c.Delete(ctx, &pv)
			Expect(err).NotTo(HaveOccurred())
		}
		time.Sleep(5000 * time.Millisecond)
	})

	AfterEach(func() {
		teardown(c)
		time.Sleep(100 * time.Millisecond)
	})

	It("should delete PersistentVolume", func() {
		pv := newHostPathPV()
		err := c.Create(ctx, pv)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() error {
			err = c.Get(ctx, client.ObjectKey{Namespace: "test", Name: "test-pv-test"}, pv)
			if err != nil {
				return err
			}

			pv.Status.Phase = corev1.VolumeReleased
			pv.Spec.ClaimRef = &corev1.ObjectReference{
				APIVersion: "v1",
				Kind:       "PersistentVolumeClaim",
				Name:       "test-pvc-test",
				Namespace:  "test",
			}

			err = c.Update(ctx, pv)
			return err
		}).Should(Succeed())

		pvc := newPVC()
		err = c.Create(ctx, pvc)
		Expect(err).NotTo(HaveOccurred())
		err = c.Get(ctx, client.ObjectKey{Namespace: "test", Name: "test-pvc-test"}, pvc)
		Expect(err).NotTo(HaveOccurred())

		err = c.Delete(ctx, pvc)
		Expect(err).NotTo(HaveOccurred())

		time.Sleep(5000 * time.Millisecond)

		Eventually(func() error {
			return c.Get(ctx, client.ObjectKey{Namespace: "test", Name: "test-pv-test"}, pv)
		}).WithTimeout(30 * time.Second).Should(HaveOccurred())

	})
})

func prepare(ctx context.Context) client.Client {
	if os.Getenv("E2E_CONTEXT") == "" {
		panic("set E2E_CONTEXT")
	}
	_, err := exec.CommandContext(ctx, "kubectl", "ctx", os.Getenv("E2E_CONTEXT")).Output()
	if err != nil {
		panic(err)
	}

	if os.Getenv("E2E_NAMESPACE") == "kind-kind" {
		_, err = exec.CommandContext(ctx, "kind", "load", "docker-image", "controller:latest").Output()
		if err != nil {
			panic(err)
		}
	}

	_, err = exec.CommandContext(ctx, "make", "-C", "../../", "install").Output()
	if err != nil {
		panic(err)
	}
	_, err = exec.CommandContext(ctx, "make", "-C", "../../", "deploy", "IMG=controller:latest").Output()
	if err != nil {
		panic(err)
	}
	cfg, err := config.GetConfigWithContext(os.Getenv("E2E_CONTEXT"))
	if err != nil {
		panic(err)
	}

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(namespacedpvv1.AddToScheme(scheme))
	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		panic(err)
	}
	ns := newTestNameSpace()

	Eventually(func() error {
		return c.Create(ctx, ns)
	}).Should(Succeed())

	return c
}

func teardown(c client.Client) {
	ctx := context.Background()
	ns := newTestNameSpace()
	c.Delete(ctx, ns, &client.DeleteOptions{})
}

func newTestNameSpace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}
}

func newHostPathPV() *corev1.PersistentVolume {
	volumeMode := corev1.PersistentVolumeFilesystem
	// nfsはtestしにくいのでテストケースではhostpathを使う
	return &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pv-test",
			Labels: map[string]string{
				"owner":           "test",
				"owner-namespace": "test",
			},
			Annotations: map[string]string{
				"pv.kubernetes.io/provisioned-by": "namespaced-pv-controller",
			},

			Finalizers: []string{
				"namespacedpv.homi.run/pvFinalizer",
			},
		},
		Spec: corev1.PersistentVolumeSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			Capacity:    corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("1Gi")},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/mnt/data",
				},
			},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
			StorageClassName:              "test-storageclass",
			VolumeMode:                    &volumeMode,
		},
	}
}
