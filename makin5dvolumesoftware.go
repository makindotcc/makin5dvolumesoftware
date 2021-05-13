package main

import (
    "github.com/brutella/hc"
    "github.com/brutella/hc/accessory"
    "github.com/brutella/hc/characteristic"
    "github.com/brutella/hc/service"
    "github.com/itchyny/volume-go"
    "log"
    "os/user"
    "time"
)

const desktopVolumeUpdateDelay = 500 * time.Millisecond

var speaker SpeakerAccessory
var desktopVolume int

type SpeakerService struct {
    *service.Service

    On     *characteristic.On
    Volume *characteristic.Brightness
}

func NewSpeakerService() *SpeakerService {
    svc := SpeakerService{}
    svc.Service = service.New(service.TypeLightbulb)

    svc.On = characteristic.NewOn()
    svc.AddCharacteristic(svc.On.Characteristic)

    svc.Volume = characteristic.NewBrightness()
    svc.AddCharacteristic(svc.Volume.Characteristic)

    return &svc
}

type SpeakerAccessory struct {
    *accessory.Accessory
    Service *SpeakerService
}

func NewSpeakerAccessory(info accessory.Info) SpeakerAccessory {
    speaker := SpeakerAccessory{}
    speaker.Accessory = accessory.New(info, accessory.TypeLightbulb)
    speaker.Service = NewSpeakerService()
    speaker.AddService(speaker.Service.Service)
    return speaker
}

func setDesktopVolume(volumeLevel int) {
    if volumeLevel == desktopVolume {
        return
    }

    log.Println("Changing desktop volume to", volumeLevel)
    desktopVolume = volumeLevel
    err := volume.SetVolume(volumeLevel)
    if err != nil {
        log.Printf("Set desktop volume to %d failed due err: %s", volumeLevel, err)
    }
}

func setHomeKitVolume(volumeLevel int) {
    speakerVolume := speaker.Service.Volume
    if speakerVolume.Value == volumeLevel {
        return
    }

    log.Println("Changing homekit volume to", volumeLevel)
    speakerVolume.SetValue(volumeLevel)
}

func createSpeaker(speakerName string) {
    speaker = NewSpeakerAccessory(accessory.Info{Name: speakerName})

    volumeLevel, err := volume.GetVolume()
    if err != nil {
        log.Fatalf("Get volume failed: %s", err)
    }
    log.Printf("Current volume: %d\n", volumeLevel)

    var enabled bool
    if volumeLevel > 0 {
        enabled = true
    } else {
        enabled = false
    }
    speaker.Service.On.SetValue(enabled)
    speaker.Service.Volume.SetValue(volumeLevel)
}

func installHomeKitVolumeUpdateListener() {
    speaker.Service.Volume.OnValueRemoteUpdate(func(remoteVolume int) {
        setDesktopVolume(remoteVolume)
    })
}

func listenForDesktopVolumeChange() {
    var err error
    desktopVolume, err = volume.GetVolume()
    if err != nil {
        log.Panicf("Listen for desktop volume change failed: %s", err)
    }
    for {
        time.Sleep(desktopVolumeUpdateDelay)
        newVolume, err := volume.GetVolume()
        if err != nil {
            log.Panicf("Listen for desktop volume change failed: %s", err)
        }
        if desktopVolume == newVolume {
            continue
        }
        desktopVolume = newVolume

        setHomeKitVolume(newVolume)
    }
}

func listenForHomeKitVolumeChange(pin string) {
    config := hc.Config{Pin: pin}
    t, err := hc.NewIPTransport(config, speaker.Accessory)
    if err != nil {
        log.Fatalf("hc new ip transport failed: %s", err)
    }

    hc.OnTermination(func() {
        <-t.Stop()
    })
    t.Start()
}

func main() {
    currentUser, err := user.Current()
    if err != nil {
        log.Fatalf("Get current user failed: %s", err)
    }
    userName := currentUser.Name
    log.Printf("Creating %s's speaker accessory", userName)

    createSpeaker(userName + "'s macbook speaker volume")
    installHomeKitVolumeUpdateListener()
    go listenForDesktopVolumeChange()
    listenForHomeKitVolumeChange("12341234")
}
