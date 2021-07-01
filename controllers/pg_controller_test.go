package controllers

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"os"
	"time"

	"github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("PG Controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		namespace = "default"

		timeout  = time.Minute * 20
		interval = time.Second * 10
	)

	var (
		pg          *v1alpha1.PG
		serviceName string
		ctx         context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		serviceName = "k8s-test-pg-acc-" + generateRandomID()
		pg = pgSpec(serviceName, namespace)

		By("Creating a new PG CR instance")
		Expect(k8sClient.Create(ctx, pg)).Should(Succeed())

		pgLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}
		createdPG := &v1alpha1.PG{}
		// We'll need to retry getting this newly created PG,
		// given that creation may not immediately happen.
		By("by retrieving PG instance from k8s")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, pgLookupKey, createdPG)

			return err == nil
		}, timeout, interval).Should(BeTrue())

		By("by waiting PG service status to become RUNNING")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, pgLookupKey, createdPG)
			if err == nil {
				return meta.IsStatusConditionTrue(createdPG.Status.Conditions, conditionTypeRunning)
			}
			return false
		}, timeout, interval).Should(BeTrue())

		By("by checking finalizers")
		Expect(createdPG.GetFinalizers()).ToNot(BeEmpty())
	})

	Context("Validating PG reconciler behaviour", func() {
		It("should createOrUpdate a new PG service", func() {
			createdPG := &v1alpha1.PG{}
			pgLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}

			Expect(k8sClient.Get(ctx, pgLookupKey, createdPG)).Should(Succeed())

			By("by checking that after creation of a PG service secret is created")
			createdSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, createdSecret)).Should(Succeed())

			Expect(createdSecret.Data["PGHOST"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PGPORT"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PGDATABASE"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PGUSER"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PGPASSWORD"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PGSSLMODE"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["DATABASE_URI"]).NotTo(BeEmpty())
		})
	})

	AfterEach(func() {
		By("Ensures that PG instance was deleted")
		ensureDelete(ctx, pg)
	})
})

func pgSpec(serviceName, namespace string) *v1alpha1.PG {
	return &v1alpha1.PG{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "PG",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: v1alpha1.PGSpec{
			ServiceCommonSpec: v1alpha1.ServiceCommonSpec{
				Project:   os.Getenv("AIVEN_PROJECT_NAME"),
				Plan:      "business-4",
				CloudName: "google-europe-west1",
			},
			PGUserConfig: v1alpha1.PGUserConfig{
				PgVersion: "12",
				PublicAccess: v1alpha1.PublicAccessUserConfig{
					Pg:         boolPointer(true),
					Prometheus: boolPointer(true),
				},
				Pg: v1alpha1.PGSubPGUserConfig{
					IdleInTransactionSessionTimeout: int64Pointer(900),
				},
			},
			AuthSecretRef: v1alpha1.AuthSecretReference{
				Name: secretRefName,
				Key:  secretRefKey,
			},
		},
	}
}