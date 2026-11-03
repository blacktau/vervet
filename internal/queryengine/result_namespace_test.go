package queryengine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestKeepsDocumentsWhole(t *testing.T) {
	stage := func(name string) bson.D { return bson.D{{Key: name, Value: bson.D{}}} }
	cases := []struct {
		name     string
		pipeline any
		want     bool
	}{
		{"empty pipeline", bson.A{}, true},
		{"selecting and ordering stages", bson.A{stage("$match"), stage("$sort"), stage("$skip"), stage("$limit")}, true},
		{"$group makes new documents", bson.A{stage("$match"), stage("$group")}, false},
		{"$project hides fields", bson.A{stage("$project")}, false},
		{"$unset hides fields", bson.A{stage("$unset")}, false},
		{"$addFields adds fields", bson.A{stage("$addFields")}, false},
		{"$lookup adds fields", bson.A{stage("$lookup")}, false},
		{"$unionWith reads another collection", bson.A{stage("$unionWith")}, false},
		{"stage with two keys is not one we know", bson.A{bson.D{{Key: "$match", Value: 1}, {Key: "$group", Value: 1}}}, false},
		{"not a pipeline", "$match", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, keepsDocumentsWhole(c.pipeline))
		})
	}
}

func TestFindNamespaceWholeDocuments(t *testing.T) {
	projection := bson.D{{Key: "name", Value: 1}}
	cases := []struct {
		name       string
		projection any
		extras     findExtras
		want       bool
	}{
		{"no projection", nil, findExtras{}, true},
		{"empty projection", bson.D{}, findExtras{}, true},
		{"projection", projection, findExtras{}, false},
		{"returnKey", nil, findExtras{ReturnKey: true}, false},
		{"showRecordId", nil, findExtras{ShowRecordID: true}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ns := findNamespace("shop", "orders", c.projection, c.extras)
			assert.Equal(t, "shop", ns.Database)
			assert.Equal(t, "orders", ns.Collection)
			assert.Equal(t, c.want, ns.WholeDocuments)
		})
	}
}
