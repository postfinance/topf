machine:
  disks:
    - device: {{ .Node.Data.storageDevice }}
      partitions:
        - mountpoint: /var/mnt/storage
