package netem

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/shinespb/pumba/pkg/chaos"
	"github.com/shinespb/pumba/pkg/container"
	"github.com/shinespb/pumba/pkg/util"

	log "github.com/sirupsen/logrus"
)

// LossCommand `netem loss` command
type LossCommand struct {
	client      container.Client
	names       []string
	pattern     string
	iface       string
	ips         []*net.IPNet
	port        uint16
	duration    time.Duration
	percent     float64
	correlation float64
	image       string
	pull        bool
	limit       int
	dryRun      bool
}

// NewLossCommand create new netem loss command
func NewLossCommand(client container.Client,
	names []string, // containers
	pattern string, // re2 regex pattern
	iface string, // network interface
	ipsList []string, // list of target ips
	port    uint16,	//destination port
	durationStr string, // chaos duration
	intervalStr string, // repeatable chaos interval
	percent float64, // loss percent
	correlation float64, // loss correlation
	image string, // traffic control image
	pull bool, // pull tc image
	limit int, // limit chaos to containers
	dryRun bool, // dry-run do not netem just log
) (chaos.Command, error) {
	// log error
	var err error
	defer func() {
		if err != nil {
			log.WithError(err).Error("failed to construct Netem Loss Command")
		}
	}()

	// get interval
	interval, err := util.GetIntervalValue(intervalStr)
	if err != nil {
		return nil, err
	}
	// get duration
	duration, err := util.GetDurationValue(durationStr, interval)
	if err != nil {
		return nil, err
	}
	// protect from Command Injection, using Regexp
	reInterface := regexp.MustCompile("[a-zA-Z][a-zA-Z0-9_-]*")
	validIface := reInterface.FindString(iface)
	if iface != validIface {
		err = fmt.Errorf("bad network interface name: must match '%s'", reInterface.String())
		return nil, err
	}
	// validate ips
	var ips []*net.IPNet
	for _, str := range ipsList {
		ip := util.ParseCIDR(str)
		if ip == nil {
			err = fmt.Errorf("bad target: '%s' is not a valid IP", str)
			return nil, err
		}
		ips = append(ips, ip)
	}
	// get netem loss percent
	if percent < 0.0 || percent > 100.0 {
		err = errors.New("invalid loss percent: must be between 0.0 and 100.0")
		return nil, err
	}
	// get netem loss variation
	if correlation < 0.0 || correlation > 100.0 {
		err = errors.New("invalid loss correlation: must be between 0.0 and 100.0")
		return nil, err
	}

	return &LossCommand{
		client:      client,
		names:       names,
		pattern:     pattern,
		iface:       iface,
		ips:         ips,
		port:		 port,
		duration:    duration,
		percent:     percent,
		correlation: correlation,
		image:       image,
		pull:        pull,
		limit:       limit,
		dryRun:      dryRun,
	}, nil
}

// Run netem loss command
func (n *LossCommand) Run(ctx context.Context, random bool) error {
	log.Debug("adding network random packet loss to all matching containers")
	log.WithFields(log.Fields{
		"names":   n.names,
		"pattern": n.pattern,
		"limit":   n.limit,
	}).Debug("listing matching containers")
	containers, err := container.ListNContainers(ctx, n.client, n.names, n.pattern, n.limit)
	if err != nil {
		log.WithError(err).Error("failed to list containers")
		return err
	}
	if len(containers) == 0 {
		log.Warning("no containers found")
		return nil
	}

	// select single random container from matching container and replace list with selected item
	if random {
		log.Debug("selecting single random container")
		if c := container.RandomContainer(containers); c != nil {
			containers = []container.Container{*c}
		}
	}

	// prepare netem loss command
	netemCmd := []string{"loss", strconv.FormatFloat(n.percent, 'f', 2, 64)}
	if n.correlation > 0 {
		netemCmd = append(netemCmd, strconv.FormatFloat(n.correlation, 'f', 2, 64))
	}

	// run netem loss command for selected containers
	var wg sync.WaitGroup
	errors := make([]error, len(containers))
	cancels := make([]context.CancelFunc, len(containers))
	for i, c := range containers {
		log.WithFields(log.Fields{
			"container": c,
		}).Debug("adding network random packet loss for container")
		netemCtx, cancel := context.WithTimeout(ctx, n.duration)
		cancels[i] = cancel
		wg.Add(1)
		go func(i int, c container.Container) {
			defer wg.Done()
			errors[i] = runNetem(netemCtx, n.client, c, n.iface, netemCmd, n.ips, n.port, n.duration, n.image, n.pull, n.dryRun)
			if errors[i] != nil {
				log.WithError(errors[i]).Error("failed to set packet loss for container")
			}
		}(i, c)
	}

	// Wait for all netem delay commands to complete
	wg.Wait()

	// cancel context to avoid leaks
	defer func() {
		for _, cancel := range cancels {
			cancel()
		}
	}()

	// scan through all errors in goroutines
	for _, e := range errors {
		// take first found error
		if e != nil {
			err = e
			break
		}
	}

	return err
}
