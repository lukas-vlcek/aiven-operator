package controllers

import (
	"context"
	"github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"time"
)

var _ = Describe("KafkaConnect Controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		namespace = "default"

		timeout  = time.Minute * 20
		interval = time.Second * 10
	)

	var (
		kafkaconnect *v1alpha1.KafkaConnect
		serviceName  string
		ctx          context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		serviceName = "k8s-test-kafkaconnect-acc-" + generateRandomID()
		kafkaconnect = kafkaConnectSpec(serviceName, namespace)

		By("Creating a new KafkaConnect CR instance")
		Expect(k8sClient.Create(ctx, kafkaconnect)).Should(Succeed())

		kcLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}
		createdKafkaConnect := &v1alpha1.KafkaConnect{}
		// We'll need to retry getting this newly created KafkaConnect,
		// given that creation may not immediately happen.
		By("by retrieving Kafka Connect instance from k8s")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, kcLookupKey, createdKafkaConnect)

			return err == nil
		}, timeout, interval).Should(BeTrue())

		By("by waiting Kafka Connect service status to become RUNNING")
		Eventually(func() string {
			err := k8sClient.Get(ctx, kcLookupKey, createdKafkaConnect)
			if err == nil {
				return createdKafkaConnect.Status.State
			}

			return ""
		}, timeout, interval).Should(Equal("RUNNING"))

		By("by checking finalizers")
		Expect(createdKafkaConnect.GetFinalizers()).ToNot(BeEmpty())
	})

	Context("Validating KafkaConnect reconciler behaviour", func() {
		It("should create a new Kafka Connect service", func() {
			createdKafkaConnect := &v1alpha1.KafkaConnect{}
			kcLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}

			Expect(k8sClient.Get(ctx, kcLookupKey, createdKafkaConnect)).Should(Succeed())

			// Let's make sure our KafkaConnect status was properly populated.
			By("by checking that after creation KafkaConnect service status fields were properly populated")
			Expect(createdKafkaConnect.Status.ServiceName).Should(Equal(serviceName))
			Expect(createdKafkaConnect.Status.State).Should(Equal("RUNNING"))
			Expect(createdKafkaConnect.Status.Plan).Should(Equal("business-4"))
			Expect(createdKafkaConnect.Status.CloudName).Should(Equal("google-europe-west1"))
			Expect(createdKafkaConnect.Status.MaintenanceWindowDow).NotTo(BeEmpty())
			Expect(createdKafkaConnect.Status.MaintenanceWindowTime).NotTo(BeEmpty())
		})
	})

	AfterEach(func() {
		By("Ensures that KafkaConnect instance was deleted")
		ensureDelete(ctx, kafkaconnect)
	})
})

func kafkaConnectSpec(serviceName, namespace string) *v1alpha1.KafkaConnect {
	return &v1alpha1.KafkaConnect{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "k8s-operator.aiven.io/v1alpha1",
			Kind:       "KafkaConnect",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: v1alpha1.KafkaConnectSpec{
			ServiceCommonSpec: v1alpha1.ServiceCommonSpec{
				Project:     os.Getenv("AIVEN_PROJECT_NAME"),
				ServiceName: serviceName,
				Plan:        "business-4",
				CloudName:   "google-europe-west1",
			},
		},
	}
}
