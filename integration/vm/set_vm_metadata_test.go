package vm_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	"k8s.io/client-go/pkg/api/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	testHelper "github.ibm.com/Bluemix/kubernetes-cpi/integration/test_assets"
)

var _ = Describe("set_vm_metadata", func() {
	var (
		clusterName                     string
		kubeConfig                      string
		rootTemplatePath, tmpConfigPath string
		replacementMap                  map[string]string
		resultOutput                    map[string]interface{}
		err                             error

		agentId string
	)

	CreateVM := func() {
		var numberOfPods int

		jsonPayload, err := testHelper.GenerateCpiJsonPayload("create_vm", rootTemplatePath, replacementMap)
		Expect(err).ToNot(HaveOccurred())

		outputBytes, err := testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
		Expect(err).ToNot(HaveOccurred())

		err = json.Unmarshal(outputBytes, &resultOutput)
		Expect(err).ToNot(HaveOccurred())
		Expect(resultOutput["result"]).ToNot(BeNil())
		Expect(resultOutput["error"]).To(BeNil())

		id := resultOutput["result"].(string)
		Expect(id).Should(ContainSubstring(clusterName))
		Expect(id).Should(ContainSubstring(agentId))
		Expect(err).ToNot(HaveOccurred())

		numberOfPods, err = testHelper.PodCount("integration")
		Expect(err).NotTo(HaveOccurred())
		Expect(numberOfPods).To(Equal(1))
	}

	BeforeEach(func() {
		clusterName = os.Getenv("CLUSTER_NAME")
		Expect(err).ToNot(HaveOccurred())

		kubeConfig = os.Getenv("KUBECONFIG")
		Expect(err).ToNot(HaveOccurred())

		pwd, err := os.Getwd()
		Expect(err).ToNot(HaveOccurred())
		rootTemplatePath = filepath.Join(pwd, "..", "..")

		agentId = "4f3d38a2-810d-4fd0-8c6a-b7dfdd614bd7"
		replacementMap = map[string]string{
			"agentID": agentId,
			"context": clusterName,
		}

		tmpConfigPath, err = testHelper.CreateTmpConfigFile(rootTemplatePath, configPath, kubeConfig)
		Expect(err).ToNot(HaveOccurred())

		CreateVM()
	})

	AfterEach(func() {
		deleteAll := exec.Command("kubectl", "-n", "integration", "delete", "po,svc,secret", "--all")
		err = deleteAll.Run()
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(func() int {
			pc, _ := testHelper.PodCount("integration")
			return pc
		}, "20s").Should(Equal(0))

		deleteCM := exec.Command("kubectl", "delete", "configmap", "--all", "-n", "integration")
		err = deleteCM.Run()
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(func() int {
			sc, _ := testHelper.ServiceCount("integration")
			return sc
		}, "20s").Should(Equal(0))
	})

	// Possible contexts: when adding new keys, when changing existing keys
	It("set the VM metadata successfully", func() {
		var pods v1.PodList
		pods, err = testHelper.GetPodListByAgentId("integration", agentId)
		Expect(err).NotTo(HaveOccurred())
		Expect(pods.Items[0].ObjectMeta.Labels["bosh.cloudfoundry.org/director"]).NotTo(Equal("bosh"))
		Expect(pods.Items[0].ObjectMeta.Labels["bosh.cloudfoundry.org/deployment"]).NotTo(Equal("cf-kube"))

		jsonPayload, err := testHelper.GenerateCpiJsonPayload("set_vm_metadata", rootTemplatePath, replacementMap)
		Expect(err).ToNot(HaveOccurred())

		outputBytes, err := testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
		Expect(err).ToNot(HaveOccurred())

		err = json.Unmarshal(outputBytes, &resultOutput)
		Expect(err).ToNot(HaveOccurred())
		Expect(resultOutput["result"]).To(BeNil())
		Expect(resultOutput["error"]).To(BeNil())

		pods, _ = testHelper.GetPodListByAgentId("integration", agentId)
		Expect(pods.Items[0].ObjectMeta.Labels["bosh.cloudfoundry.org/director"]).To(Equal("bosh"))
		Expect(pods.Items[0].ObjectMeta.Labels["bosh.cloudfoundry.org/deployment"]).To(Equal("cf-kube"))
	})
})
