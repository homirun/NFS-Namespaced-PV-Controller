package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("persistent volume controller", func() {

	ctx := context.Background()
	var stopFunc func()

	BeforeEach(func() {
		pvs := &corev1.PersistentVolumeList{}
		err := k8sClient.List(ctx, pvs)
		Expect(err).NotTo(HaveOccurred())
		for _, pv := range pvs.Items {
			err = k8sClient.Delete(ctx, &pv)
			Expect(err).NotTo(HaveOccurred())
		}
		time.Sleep(5000 * time.Millisecond)

		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme: scheme.Scheme,
		})
		Expect(err).NotTo(HaveOccurred())

		reconciler := &PersistentVolumeReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		}
		err = reconciler.SetupWithManager(mgr)
		Expect(err).NotTo(HaveOccurred())

		ctx, cancel := context.WithCancel(ctx)
		stopFunc = cancel
		go func() {
			err := mgr.Start(ctx)
			if err != nil {
				panic(err)
			}
		}()
		time.Sleep(3000 * time.Millisecond)

	})

	AfterEach(func() {
		stopFunc()
		time.Sleep(100 * time.Millisecond)
	})

	// It("should delete PersistentVolume", func() {
	// 	pv := newPV()
	// 	err := k8sClient.Create(ctx, pv)
	// 	Expect(err).NotTo(HaveOccurred())

	// 	pvc := newPVC()
	// 	err = k8sClient.Create(ctx, pvc)
	// 	Expect(err).NotTo(HaveOccurred())

	// 	pv.Status.Phase = corev1.VolumeReleased
	// 	// climeRefを設定する
	// 	pv.Spec.ClaimRef = &corev1.ObjectReference{
	// 		APIVersion: "v1",
	// 		Kind:       "PersistentVolumeClaim",
	// 		Name:       "test-pvc-test",
	// 		Namespace:  "test",
	// 	}

	// 	// ここで再度finalizerを上書きしないとkubernetes.io/pv-protectionが残る
	// 	// pv.ObjectMeta.Finalizers = []string{"namespacedpv.homi.run/pvFinalizer"}
	// 	err = k8sClient.Update(ctx, pv)
	// 	Expect(err).NotTo(HaveOccurred())

	// 	err = k8sClient.Get(ctx, client.ObjectKey{Namespace: "test", Name: "test-pvc-test"}, pvc)
	// 	Expect(err).NotTo(HaveOccurred())
	// 	err = k8sClient.Delete(ctx, pvc)
	// 	Expect(err).NotTo(HaveOccurred())

	// 	time.Sleep(5000 * time.Millisecond)

	// 	Eventually(func() error {
	// 		return k8sClient.Get(ctx, client.ObjectKey{Namespace: "test", Name: "test-pv-test"}, pv)
	// 	}).Should(Succeed())
	// 	Expect(pv).To(BeNil())

	// })

})

func newPV() *corev1.PersistentVolume {
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

func newPVC() *corev1.PersistentVolumeClaim {
	storageClass := "test-storageclass"
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc-test",
			Namespace: "test",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
			StorageClassName: &storageClass,
		},
	}
}
