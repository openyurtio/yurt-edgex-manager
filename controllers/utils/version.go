package util

import (
	"context"
	"fmt"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

const IotCtrlName = "edgex-controller-manager"

func DefaultVersion(ctx context.Context, cli client.Client) (string, string, error) {
	var list v1.DeploymentList
	err := wait.PollImmediate(5*time.Second, 15*time.Second, func() (done bool, err error) {
		s := labels.SelectorFromSet(labels.Set{"control-plane": IotCtrlName})

		err = cli.List(ctx, &list, client.MatchingLabelsSelector{Selector: s})
		if err != nil {
			klog.Errorf("failed to get deploy %s ,err:%v", IotCtrlName, err)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return "", "", err
	}

	if len(list.Items) > 1 {
		return "", "", fmt.Errorf("more than one %s exist", IotCtrlName)
	}

	var iotCtrlImage string
	cntrs := list.Items[0].Spec.Template.Spec.Containers
	for _, cntr := range cntrs {
		if cntr.Name == "manager" {
			iotCtrlImage = cntr.Image
		}
	}

	version := iotCtrlImage[strings.LastIndex(iotCtrlImage, ":")+1:]
	ns := list.Items[0].Namespace
	klog.Infof("default version: %s, default namespace", version, ns)
	return version, ns, nil
}
