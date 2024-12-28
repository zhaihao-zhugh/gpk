package mongodb

import (
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

type Time time.Time

func (t *Time) UnmarshalJSON(data []byte) (err error) {
	now, err := time.ParseInLocation(time.DateTime, string(data), time.Local)
	*t = Time(now)
	return
}

func (t Time) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len(time.DateTime)+2)
	b = append(b, '"')
	b = time.Time(t).AppendFormat(b, time.DateTime)
	b = append(b, '"')
	return b, nil
}

func (t *Time) MarshalBSONValue() (bsontype.Type, []byte, error) {
	targetTime := primitive.NewDateTimeFromTime(time.Time(*t))
	return bson.MarshalValue(targetTime)
}

func (t *Time) UnmarshalBSONValue(bt bsontype.Type, data []byte) error {
	v, _, valid := bsoncore.ReadValue(data, bt)
	if !valid {
		return errors.New(fmt.Sprintf("读取数据失败：%s %s", bt, data))
	}
	if v.Type == bsontype.DateTime {
		*t = Time(v.Time())
	}
	return nil
}

func GetTime(t time.Time) Time {
	return Time(t)
}
