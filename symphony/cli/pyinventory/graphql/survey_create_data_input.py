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

from .survey_status_enum import SurveyStatus

from .survey_question_response_input import SurveyQuestionResponse


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
class SurveyCreateData:
    name: str
    completionTimestamp: int
    locationID: str
    surveyResponses: List[SurveyQuestionResponse]
    ownerName: Optional[str] = None
    creationTimestamp: Optional[int] = None
    status: Optional[SurveyStatus] = None

