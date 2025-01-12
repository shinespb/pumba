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

// LossGECommand `netem loss gemodel` (Gilbert-Elliot model) command
type LossGECommand struct {
	client   container.Client
	names    []string
	pattern  string
	iface    string
	ips      []*net.IPNet
	port     uint16
	duration time.Duration
	pg       float64
	pb       float64
	oneH     float64
	oneK     float64
	image    string
	pull     bool
	limit    int
	dryRun   bool
}

// NewLossGECommand create new netem loss gemodel (Gilbert-Elliot) command
func NewLossGECommand(client container.Client,
	names []string, // containers
	pattern string, // re2 regex pattern
	iface string, // network interface
	ipsList []string, // list of target ips
	port    uint16,
	durationStr string, // chaos duration
	intervalStr string, // repeatable chaos interval
	pg float64, // Good State transition probability
	pb float64, // Bad State transition probability
	oneH float64, // loss probability in Bad state
	oneK float64, // loss probability in Good state
	image string, // traffic control image
	pull bool, // pull tc image
	limit int, // limit chaos to containers
	dryRun bool, // dry-run do not netem just log
) (chaos.Command, error) {
	// log error
	var err error
	defer func() {
		if err != nil {
			log.WithError(err).Error("failed to construct Netem Loss GEModel Command")
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
	// get pg - Good State transition probability
	if pg < 0.0 || pg > 100.0 {
		err = errors.New("Invalid pg (Good State) transition probability: must be between 0.0 and 100.0")
		log.Error(err)
		return nil, err
	}
	// get pb - Bad State transition probability
	if pb < 0.0 || pb > 100.0 {
		err = errors.New("Invalid pb (Bad State) transition probability: must be between 0.0 and 100.0")
		log.Error(err)
		return nil, err
	}
	// get (1-h) - loss probability in Bad state
	if oneH < 0.0 || oneH > 100.0 {
		err = errors.New("Invalid loss probability: must be between 0.0 and 100.0")
		log.Error(err)
		return nil, err
	}
	// get (1-k) - loss probability in Good state
	if oneK < 0.0 || oneK > 100.0 {
		err = errors.New("Invalid loss probability: must be between 0.0 and 100.0")
		log.Error(err)
		return nil, err
	}

	return &LossGECommand{
		client:   client,
		names:    names,
		pattern:  pattern,
		iface:    iface,
		ips:      ips,
		port:	  port,
		duration: duration,
		pg:       pg,
		pb:       pb,
		oneH:     oneH,
		oneK:     oneK,
		image:    image,
		pull:     pull,
		limit:    limit,
		dryRun:   dryRun,
	}, nil
}

// Run netem loss state command
func (n *LossGECommand) Run(ctx context.Context, random bool) error {
	log.Debug("adding network packet loss according Gilbert-Elliot model to all matching containers")
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

	// prepare netem loss gemodel command
	netemCmd := []string{"loss", "gemodel", strconv.FormatFloat(n.pg, 'f', 2, 64)}
	netemCmd = append(netemCmd, strconv.FormatFloat(n.pb, 'f', 2, 64))
	netemCmd = append(netemCmd, strconv.FormatFloat(n.oneH, 'f', 2, 64))
	netemCmd = append(netemCmd, strconv.FormatFloat(n.oneK, 'f', 2, 64))

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
