local config = require 'pl.config'
local stringio = require 'pl.stringio'
asserteq = require 'pl.test'.asserteq

function testconfig(test,tbl,cfg)
    local f = stringio.open(test)
    local c = config.read(f,cfg)
    f:close()
    if not tbl then
        print(pretty.write(c))
    else
        asserteq(c,tbl)
    end
end

testconfig ([[
 ; comment 2 (an ini file)
[section!]
bonzo.dog=20,30
config_parm=here we go again
depth = 2
[another]
felix="cat"
]],{
  section_ = {
    bonzo_dog = { -- comma-sep values get split by default
      20,
      30
    },
    depth = 2,
    config_parm = "here we go again"
  },
  another = {
    felix = "\"cat\""
  }
})


testconfig ([[
# this is a more Unix-y config file
fred = 1
alice = 2
home.dog = /bonzo/dog/etc
]],{
  home_dog = "/bonzo/dog/etc",  -- note the default is {variablilize = true}
  fred = 1,
  alice = 2
})

-- backspace line continuation works, thanks to config.lines function
testconfig ([[
foo=frodo,a,c,d, \
  frank, alice, boyo
]],
{
  foo = {
    "frodo",
    "a",
    "c",
    "d",
    "frank",
    "alice",
    "boyo"
  }
}
)

------ options to control default behaviour -----

-- want to keep key names as is!
testconfig ([[
alpha.dog=10
# comment here
]],{
    ["alpha.dog"]=10
},{variabilize=false})

-- don't convert strings to numbers
testconfig ([[
alpha.dog=10
; comment here
]],{
    alpha_dog="10"
},{convert_numbers=false})

-- convert strings to booleans
testconfig ([[
alpha.dog=false
alpha.cat=true
; comment here
]],{
    alpha_dog=false,
    alpha_cat=true
},{convert_boolean=true})

-- don't split comma-lists by setting the list delimiter to something else
testconfig ([[
extra=10,'hello',42
]],{
    extra="10,'hello',42"
},{list_delim='@'})

-- Unix-style password file
testconfig([[
lp:x:7:7:lp:/var/spool/lpd:/bin/sh
mail:x:8:8:mail:/var/mail:/bin/sh
news:x:9:9:news:/var/spool/news:/bin/sh
]],
{
  {
    "lp",
    "x",
    7,
    7,
    "lp",
    "/var/spool/lpd",
    "/bin/sh"
  },
  {
    "mail",
    "x",
    8,
    8,
    "mail",
    "/var/mail",
    "/bin/sh"
  },
  {
    "news",
    "x",
    9,
    9,
    "news",
    "/var/spool/news",
    "/bin/sh"
  }
},
{list_delim=':'})

-- Unix updatedb.conf is in shell script form, but config.read
-- copes by extracting the variables as keys and the export
-- commands as the array part; there is an option to remove quotes
-- from values
testconfig([[
# Global options for invocations of find(1)
FINDOPTIONS='-ignore_readdir_race'
export FINDOPTIONS
]],{
  "export FINDOPTIONS",
  FINDOPTIONS = "-ignore_readdir_race"
},{trim_quotes=true})

-- Unix fstab format. No key/value assignments so use `ignore_assign`;
-- list values are separated by a number of spaces
testconfig([[
# <file system> <mount point>   <type>  <options>       <dump>  <pass>
proc            /proc           proc    defaults        0       0
/dev/sda1       /               ext3    defaults,errors=remount-ro 0       1
]],
{
  {
    "proc",
    "/proc",
    "proc",
    "defaults",
    0,
    0
  },
  {
    "/dev/sda1",
    "/",
    "ext3",
    "defaults,errors=remount-ro",
    0,
    1
  }
},
{list_delim='%s+',ignore_assign=true}
)

-- Linux procfs 'files' often use ':' as the key/pair separator;
-- a custom convert_numbers handles the units properly!
-- Here is the first two lines from /proc/meminfo
testconfig([[
MemTotal:        1024748 kB
MemFree:          220292 kB
]],
{ MemTotal = 1024748, MemFree = 220292 },
{
 keysep = ':',
 convert_numbers = function(s)
    s = s:gsub(' kB$','')
    return tonumber(s)
  end
 }
)

-- altho this works, rather use pl.data.read for this kind of purpose.
testconfig ([[
# this is just a set of comma-separated values
1000,444,222
44,555,224
]],{
  {
    1000,
    444,
    222
  },
  {
    44,
    555,
    224
  }
})

--- new with 1.0.3: smart configuration file reading
-- handles a number of common Unix file formats automatically

function smart(f)
    f = stringio.open(f)
    return config.read(f,{smart=true})
end

-- /etc/fstab
asserteq (smart[[
# /etc/fstab: static file system information.
#
# Use 'blkid -o value -s UUID' to print the universally unique identifier
# for a device; this may be used with UUID= as a more robust way to name
# devices that works even if disks are added and removed. See fstab(5).
#
# <file system> <mount point>   <type>  <options>       <dump>  <pass>
proc            /proc           proc    nodev,noexec,nosuid 0       0
/dev/sdb2       /               ext2    errors=remount-ro 0       1
/dev/fd0        /media/floppy0  auto    rw,user,noauto,exec,utf8 0       0
]],{
  proc = {
    "/proc",
    "proc",
    "nodev,noexec,nosuid",
    0,
    0
  },
  ["/dev/sdb2"] = {
    "/",
    "ext2",
    "errors=remount-ro",
    0,
    1
  },
  ["/dev/fd0"] = {
    "/media/floppy0",
    "auto",
    "rw,user,noauto,exec,utf8",
    0,
    0
  }
})

-- /proc/XXXX/status
asserteq (smart[[
Name:	bash
State:	S (sleeping)
Tgid:	30071
Pid:	30071
PPid:	1587
TracerPid:	0
Uid:	1000	1000	1000	1000
Gid:	1000	1000	1000	1000
FDSize:	256
Groups:	4 20 24 46 105 119 122 1000
VmPeak:     6780 kB
VmSize:     6716 kB
]],{
  Pid = 30071,
  VmSize = 6716,
  PPid = 1587,
  Tgid = 30071,
  State = "S (sleeping)",
  Uid = "1000	1000	1000	1000",
  Name = "bash",
  Gid = "1000	1000	1000	1000",
  Groups = "4 20 24 46 105 119 122 1000",
  FDSize = 256,
  VmPeak = 6780,
  TracerPid = 0
})

-- ssh_config
asserteq (smart[[
Host *
#   ForwardAgent no
#   ForwardX11 no
#   Tunnel no
#   TunnelDevice any:any
#   PermitLocalCommand no
#   VisualHostKey no
    SendEnv LANG LC_*
    HashKnownHosts yes
    GSSAPIAuthentication yes
    GSSAPIDelegateCredentials no
]],{
  Host = "*",
  GSSAPIAuthentication = "yes",
  SendEnv = "LANG LC_*",
  HashKnownHosts = "yes",
  GSSAPIDelegateCredentials = "no"
})

-- updatedb.conf
asserteq (smart[[
PRUNE_BIND_MOUNTS="yes"
# PRUNENAMES=".git .bzr .hg .svn"
PRUNEPATHS="/tmp /var/spool /media"
PRUNEFS="NFS nfs nfs4 rpc_pipefs afs binfmt_misc proc smbfs autofs iso9660 ncpfs coda devpts ftpfs devfs mfs shfs sysfs cifs lustre_lite tmpfs usbfs udf fuse.glusterfs fuse.sshfs ecryptfs fusesmb devtmpfs"
]],{
  PRUNEPATHS = "/tmp /var/spool /media",
  PRUNE_BIND_MOUNTS = "yes",
  PRUNEFS = "NFS nfs nfs4 rpc_pipefs afs binfmt_misc proc smbfs autofs iso9660 ncpfs coda devpts ftpfs devfs mfs shfs sysfs cifs lustre_lite tmpfs usbfs udf fuse.glusterfs fuse.sshfs ecryptfs fusesmb devtmpfs"
})
