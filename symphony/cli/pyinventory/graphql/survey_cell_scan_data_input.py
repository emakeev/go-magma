#!/usr/bin/env python3
# @generated AUTOGENERATED file. Do not Change!

from dataclasses import dataclass, field
from datetime import datetime
from functools import partial
from numbers import Number
from typing import Any, Callable, List, Mapping, Optional

from dataclasses_json import dataclass_json
from marshmallow import fields as marshmallow_fields

from .datetime_utils import fromisoformat

from .cellular_network_type_enum import CellularNetworkType



DATETIME_FIELD = field(
    metadata={
        "dataclasses_json": {
            "encoder": datetime.isoformat,
            "decoder": fromisoformat,
            "mm_field": marshmallow_fields.DateTime(format="iso"),
        }
    }
)


def enum_field(enum_type):
    def encode_enum(value):
        return value.value

    def decode_enum(t, value):
        return t(value)

    return field(
        metadata={
            "dataclasses_json": {
                "encoder": encode_enum,
                "decoder": partial(decode_enum, enum_type),
            }
        }
    )


@dataclass_json
@dataclass
class SurveyCellScanData:
    networkType: CellularNetworkType = enum_field(CellularNetworkType)
    signalStrength: int
    timestamp: Optional[int] = None
    baseStationID: Optional[str] = None
    networkID: Optional[str] = None
    systemID: Optional[str] = None
    cellID: Optional[str] = None
    locationAreaCode: Optional[str] = None
    mobileCountryCode: Optional[str] = None
    mobileNetworkCode: Optional[str] = None
    primaryScramblingCode: Optional[str] = None
    operator: Optional[str] = None
    arfcn: Optional[int] = None
    physicalCellID: Optional[str] = None
    trackingAreaCode: Optional[str] = None
    timingAdvance: Optional[int] = None
    earfcn: Optional[int] = None
    uarfcn: Optional[int] = None
    latitude: Optional[Number] = None
    longitude: Optional[Number] = None

