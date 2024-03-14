package e2e

import (
	"context"
	"os"
	"os/exec"

	namespacedpvv1 "github.com/homirun/namespaced-pv-controller/api/v1"
	. "github.com/onsi/ginkgo/v2"
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
	e2e_test()
})

func e2e_test() {
	ctx := context.Background()
	client := prepare(ctx)

	teardown(client)
}

func prepare(ctx context.Context) client.Client {
	if os.Getenv("E2E_CONTEXT") == "" {
		panic("set E2E_CONTEXT")
	}
	_, err := exec.CommandContext(ctx, "kubectx", os.Getenv("E2E_CONTEXT")).Output()
	if err != nil {
		panic(err)
	}
	_, err = exec.CommandContext(ctx, "make", "-C", "..", "deploy", "IMG=localhost:5000/controller").Output()
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
	err = c.Create(ctx, ns, &client.CreateOptions{})
	if err != nil {
		panic(err)
	}

	np := newNamespacedPv()
	err = c.Create(ctx, np, &client.CreateOptions{})
	if err != nil {
		panic(err)
	}

	return c
}

func teardown(c client.Client) {
	ctx := context.Background()
	np := newNamespacedPv()
	c.Delete(ctx, np, &client.DeleteOptions{})
}

func newNamespacedPv() *namespacedpvv1.NamespacedPv {
	return &namespacedpvv1.NamespacedPv{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "namespaced-pv",
			Namespace: "test",
		},
		Spec: namespacedpvv1.NamespacedPvSpec{
			VolumeName:       "test-pv",
			StorageClassName: "test-storageclass",
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("1Gi"),
			},
			Nfs: namespacedpvv1.NFS{
				Server:   "127.0.0.1",
				Path:     "/data/share",
				ReadOnly: false,
			},
			ReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			MountOptions:  "nolock,vers=4.1",
			ClaimRefName:  "test-pvc",
		},
	}
}

func newTestNameSpace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}
}
