package smart

import (
	"testing"

	"github.com/function61/gokit/assert"
)

func TestParse(t *testing.T) {
	rep, err := parseSmartCtlJsonReport([]byte(exampleOutput))
	assert.Assert(t, err == nil)

	assert.Assert(t, rep.AtaSmartAttributes.Table[0].Name == "Raw_Read_Error_Rate")
	assert.Assert(t, rep.Temperature.Current == 36)
	assert.Assert(t, rep.SmartStatus.Passed)
	assert.Assert(t, rep.PowerCycleCount == 19)
	assert.Assert(t, rep.PowerOnTime.Hours == 1456)

	assert.Assert(t, rep.FindSmartAttributeByName("notfound") == nil)

	assert.Assert(t, rep.FindSmartAttributeByName("Raw_Read_Error_Rate").Raw.Value == 0)
	assert.Assert(t, rep.FindSmartAttributeByName("FTL_Program_Page_Count").Raw.Value == 983715840)
}

const exampleOutput = `{
  "json_format_version": [
    1,
    0
  ],
  "smartctl": {
    "version": [
      7,
      0
    ],
    "svn_revision": "4883",
    "platform_info": "x86_64-linux-5.0.0-29-generic",
    "build_info": "(local build)",
    "argv": [
      "smartctl",
      "-a",
      "-j",
      "/dev/sda"
    ],
    "exit_status": 0
  },
  "device": {
    "name": "/dev/sda",
    "info_name": "/dev/sda [SAT]",
    "type": "sat",
    "protocol": "ATA"
  },
  "model_family": "Crucial/Micron BX/MX1/2/3/500, M5/600, 1100 SSDs",
  "model_name": "CT960BX500SSD1",
  "serial_number": "1904E16F0268",
  "wwn": {
    "naa": 0,
    "oui": 0,
    "id": 0
  },
  "firmware_version": "M6CR022",
  "user_capacity": {
    "blocks": 1875385008,
    "bytes": 960197124096
  },
  "logical_block_size": 512,
  "physical_block_size": 512,
  "rotation_rate": 0,
  "form_factor": {
    "ata_value": 3,
    "name": "2.5 inches"
  },
  "in_smartctl_database": true,
  "ata_version": {
    "string": "ACS-3 T13/2161-D revision 4",
    "major_value": 2040,
    "minor_value": 283
  },
  "sata_version": {
    "string": "SATA 3.2",
    "value": 255
  },
  "interface_speed": {
    "max": {
      "sata_value": 14,
      "string": "6.0 Gb/s",
      "units_per_second": 60,
      "bits_per_unit": 100000000
    },
    "current": {
      "sata_value": 2,
      "string": "3.0 Gb/s",
      "units_per_second": 30,
      "bits_per_unit": 100000000
    }
  },
  "local_time": {
    "time_t": 1569844728,
    "asctime": "Mon Sep 30 11:58:48 2019 UTC"
  },
  "smart_status": {
    "passed": true
  },
  "ata_smart_data": {
    "offline_data_collection": {
      "status": {
        "value": 0,
        "string": "was never started"
      },
      "completion_seconds": 120
    },
    "self_test": {
      "status": {
        "value": 0,
        "string": "completed without error",
        "passed": true
      },
      "polling_minutes": {
        "short": 2,
        "extended": 10
      }
    },
    "capabilities": {
      "values": [
        17,
        2
      ],
      "exec_offline_immediate_supported": true,
      "offline_is_aborted_upon_new_cmd": false,
      "offline_surface_scan_supported": false,
      "self_tests_supported": true,
      "conveyance_self_test_supported": false,
      "selective_self_test_supported": false,
      "attribute_autosave_enabled": false,
      "error_logging_supported": true,
      "gp_logging_supported": true
    }
  },
  "ata_smart_attributes": {
    "revision": 1,
    "table": [
      {
        "id": 1,
        "name": "Raw_Read_Error_Rate",
        "value": 0,
        "worst": 100,
        "thresh": 0,
        "when_failed": "",
        "flags": {
          "value": 47,
          "string": "POSR-K ",
          "prefailure": true,
          "updated_online": true,
          "performance": true,
          "error_rate": true,
          "event_count": false,
          "auto_keep": true
        },
        "raw": {
          "value": 0,
          "string": "0"
        }
      },
      {
        "id": 5,
        "name": "Reallocate_NAND_Blk_Cnt",
        "value": 100,
        "worst": 100,
        "thresh": 10,
        "when_failed": "",
        "flags": {
          "value": 50,
          "string": "-O--CK ",
          "prefailure": false,
          "updated_online": true,
          "performance": false,
          "error_rate": false,
          "event_count": true,
          "auto_keep": true
        },
        "raw": {
          "value": 0,
          "string": "0"
        }
      },
      {
        "id": 248,
        "name": "FTL_Program_Page_Count",
        "value": 100,
        "worst": 100,
        "thresh": 0,
        "when_failed": "",
        "flags": {
          "value": 50,
          "string": "-O--CK ",
          "prefailure": false,
          "updated_online": true,
          "performance": false,
          "error_rate": false,
          "event_count": true,
          "auto_keep": true
        },
        "raw": {
          "value": 983715840,
          "string": "983715840"
        }
      }
    ]
  },
  "power_on_time": {
    "hours": 1456
  },
  "power_cycle_count": 19,
  "temperature": {
    "current": 36
  },
  "ata_smart_error_log": {
    "summary": {
      "revision": 1,
      "count": 0
    }
  },
  "ata_smart_self_test_log": {
    "standard": {
      "revision": 1,
      "count": 0
    }
  }
}
`
