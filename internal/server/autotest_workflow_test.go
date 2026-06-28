package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestRuntimeGeneratedWorkflow(t *testing.T) {
	handler := newTestHandler()

	cases := []struct {
		name           string
		mapping        string
		path           string
		requestBody    string
		wantSubstrings []string
		resetGRPC      bool
	}{
		{
			name: "pdm",
			mapping: `{
			  "name": "pdm_gen_for_contract",
			  "persistent": true,
			  "request": {
			    "urlPath": "/pdm_api_gateway.v1.MCProduct/WarehousesByNomenclature",
			    "method": "POST",
			    "bodyPatterns": [
			      {
			        "matchesJsonPath": "$[?(@.guids == ['nom-1','nom-2'])]"
			      }
			    ]
			  },
			  "response": {
			    "status": 200,
			    "body": "{\"warehouses\":[{\"warehouses_guid\":[\"warehouse-1\"],\"nomenclature_guid\":\"nom-1\"}]}",
			    "transformers": [
			      "response-template"
			    ]
			  },
			  "priority": 1,
			  "metadata": {
			    "wiremock-gui": {
			      "folder": "/generated/PDM/WarehousesByNomenclature"
			    }
			  }
			}`,
			path:           "/pdm_api_gateway.v1.MCProduct/WarehousesByNomenclature",
			requestBody:    `{"guids":["nom-1","nom-2"]}`,
			wantSubstrings: []string{`"warehouses"`, "warehouse-1"},
			resetGRPC:      true,
		},
		{
			name: "shcat",
			mapping: `{
			  "name": "shcat_gen_for_contract",
			  "persistent": true,
			  "request": {
			    "url": "/druz-shcat2/rpc",
			    "method": "POST",
			    "bodyPatterns": [
			      {
			        "matchesJsonPath": "$[?(@.method == 'rests.get')]"
			      },
			      {
			        "matchesJsonPath": "$.params.nomenclature[?(@ == 'nom-1')]"
			      },
			      {
			        "matchesJsonPath": "$.params[?(@.source.size() == 2)]"
			      },
			      {
			        "matchesJsonPath": "$.params.source[?(@ == 'warehouse-1')]"
			      },
			      {
			        "matchesJsonPath": "$[?(@.params.provider_strict == true)]"
			      }
			    ]
			  },
			  "response": {
			    "status": 200,
			    "jsonBody": {
			      "jsonrpc": "2.0",
			      "id": "{{jsonPath request.body '$.id'}}",
			      "result": [
			        {
			          "nomenclature": "nom-1",
			          "source": "warehouse-1",
			          "quantity": 5
			        }
			      ]
			    },
			    "transformers": [
			      "response-template"
			    ]
			  },
			  "priority": 1,
			  "metadata": {
			    "wiremock-gui": {
			      "folder": "/generated/ShCat2/rests.get"
			    }
			  }
			}`,
			path:           "/druz-shcat2/rpc",
			requestBody:    `{"jsonrpc":"2.0","method":"rests.get","params":{"nomenclature":["nom-1"],"source":["warehouse-1","warehouse-2"],"provider_strict":true},"id":"req-shcat"}`,
			wantSubstrings: []string{"req-shcat", "warehouse-1"},
		},
		{
			name: "officer",
			mapping: `{
			  "name": "officer_get_office_stock_info_gen_for_contract",
			  "persistent": true,
			  "request": {
			    "method": "POST",
			    "url": "/druz-officer/rpc",
			    "bodyPatterns": [
			      {
			        "matchesJsonPath": "$[?(@.method == 'availability.get_office_stock_info')]"
			      },
			      {
			        "matchesJsonPath": "$[?(@.params.office_id.size() == 2)]"
			      },
			      {
			        "matchesJsonPath": "$.params.office_id[?(@ == 'office-1')]"
			      }
			    ]
			  },
			  "response": {
			    "status": 200,
			    "jsonBody": {
			      "jsonrpc": "2.0",
			      "result": [
			        {
			          "officeId": "office-1",
			          "stock": 7
			        }
			      ],
			      "id": "{{jsonPath request.body '$.id'}}"
			    },
			    "transformers": [
			      "response-template"
			    ]
			  },
			  "priority": 1,
			  "metadata": {
			    "wiremock-gui": {
			      "folder": "/generated/Officer/availability.get_office_stock_info"
			    }
			  }
			}`,
			path:           "/druz-officer/rpc",
			requestBody:    `{"jsonrpc":"2.0","method":"availability.get_office_stock_info","params":{"office_id":["office-1","office-2"]},"id":"req-officer"}`,
			wantSubstrings: []string{"req-officer", "office-1"},
		},
		{
			name: "susanin",
			mapping: `{
			  "name": "susanin_get_logistic_chains_v3_gen_for_contract",
			  "persistent": true,
			  "request": {
			    "urlPath": "/druz-susanin/rpc",
			    "method": "POST",
			    "bodyPatterns": [
			      {
			        "matchesJsonPath": "$[?(@.method == 'get_logistic_chains_v3')]"
			      },
			      {
			        "matchesJsonPath": "$.params[?(@.destinations.size() == 2)]"
			      },
			      {
			        "matchesJsonPath": "$.params.destinations[?(@ == 'destination-1')]"
			      },
			      {
			        "matchesJsonPath": "$.params[?(@.providers.size() == 1)]"
			      },
			      {
			        "matchesJsonPath": "$.params.providers[?(@ == 'provider-1')]"
			      }
			    ]
			  },
			  "response": {
			    "status": 200,
			    "jsonBody": {
			      "jsonrpc": "2.0",
			      "result": [
			        {
			          "Route": [
			            "provider-1",
			            "destination-1"
			          ],
			          "Priority": 1
			        }
			      ],
			      "id": "{{jsonPath request.body '$.id'}}"
			    },
			    "transformers": [
			      "response-template"
			    ]
			  },
			  "priority": 1,
			  "metadata": {
			    "wiremock-gui": {
			      "folder": "/generated/Susanin/get_logistic_chains_v3"
			    }
			  }
			}`,
			path:           "/druz-susanin/rpc",
			requestBody:    `{"jsonrpc":"2.0","method":"get_logistic_chains_v3","params":{"destinations":["destination-1","destination-2"],"providers":["provider-1"]},"id":"req-susanin"}`,
			wantSubstrings: []string{"req-susanin", "provider-1"},
		},
		{
			name: "vanga",
			mapping: `{
			  "name": "vanga_schedule_predict_dates_gen_for_contract",
			  "persistent": true,
			  "request": {
			    "url": "/druz-vanga/rpc",
			    "method": "POST",
			    "bodyPatterns": [
			      {
			        "matchesJsonPath": "$[?(@.method == 'schedule.predict_dates')]"
			      },
			      {
			        "matchesJsonPath": "$.params[?(@.extended == false)]"
			      },
			      {
			        "matchesJsonPath": "$.params[?(@.chains.size() == 1)]"
			      },
			      {
			        "matchesJsonPath": "$.params.chains.*[?(@.chain_nodes == ['source-1','destination-1'])]"
			      }
			    ]
			  },
			  "response": {
			    "status": 200,
			    "jsonBody": {
			      "jsonrpc": "2.0",
			      "result": {
			        "0": {
			          "chain_nodes": [
			            "source-1",
			            "destination-1"
			          ],
			          "delivery_date": "2026-01-02"
			        }
			      },
			      "id": "{{jsonPath request.body '$.id'}}"
			    },
			    "transformers": [
			      "response-template"
			    ]
			  },
			  "priority": 1,
			  "metadata": {
			    "wiremock-gui": {
			      "folder": "/generated/Vanga/schedule.predict_dates"
			    }
			  }
			}`,
			path:           "/druz-vanga/rpc",
			requestBody:    `{"jsonrpc":"2.0","method":"schedule.predict_dates","params":{"extended":false,"chains":{"0":{"chain_nodes":["source-1","destination-1"]}}},"id":"req-vanga"}`,
			wantSubstrings: []string{"req-vanga", "2026-01-02"},
		},
		{
			name: "courier",
			mapping: `{
			  "name": "courier_get_courier_delivery_date_gen_for_contract",
			  "persistent": true,
			  "request": {
			    "url": "/druz-courier/rpc",
			    "method": "POST",
			    "bodyPatterns": [
			      {
			        "matchesJsonPath": "$[?(@.method == 'get_courier_delivery_date')]"
			      },
			      {
			        "matchesJsonPath": "$.params[0][?(@.office_id == 'office-1')]"
			      },
			      {
			        "matchesJsonPath": "$.params[0][?(@.pickup_dates.size() == 2)]"
			      },
			      {
			        "matchesJsonPath": "$.params[0].pickup_dates[?(@ == '2026-01-02')]"
			      }
			    ]
			  },
			  "response": {
			    "status": 200,
			    "jsonBody": {
			      "jsonrpc": "2.0",
			      "result": [
			        {
			          "office_id": "office-1",
			          "pickup_dates": {
			            "2026-01-02": "2026-01-03T09:00:00Z"
			          }
			        }
			      ],
			      "id": "{{jsonPath request.body '$.id'}}"
			    },
			    "transformers": [
			      "response-template"
			    ]
			  },
			  "priority": 1,
			  "metadata": {
			    "wiremock-gui": {
			      "folder": "/generated/Courier/get_courier_delivery_date"
			    }
			  }
			}`,
			path:           "/druz-courier/rpc",
			requestBody:    `{"jsonrpc":"2.0","method":"get_courier_delivery_date","params":[{"office_id":"office-1","pickup_dates":["2026-01-02","2026-01-04"]}],"id":"req-courier"}`,
			wantSubstrings: []string{"req-courier", "2026-01-03T09:00:00Z"},
		},
		{
			name: "fry",
			mapping: `{
			  "name": "fry_get_courier_office_gen_for_contract",
			  "persistent": true,
			  "request": {
			    "url": "/druz-fry/rpc",
			    "method": "POST",
			    "bodyPatterns": [
			      {
			        "matchesJsonPath": "$[?(@.method == 'get_courier_office')]"
			      },
			      {
			        "matchesJsonPath": "$.params.coords[?(@.lat == '55.75')]"
			      },
			      {
			        "matchesJsonPath": "$.params.coords[?(@.lon == '37.61')]"
			      }
			    ]
			  },
			  "response": {
			    "status": 200,
			    "jsonBody": {
			      "jsonrpc": "2.0",
			      "result": [
			        "office-1"
			      ],
			      "id": "{{jsonPath request.body '$.id'}}"
			    },
			    "transformers": [
			      "response-template"
			    ]
			  },
			  "priority": 1,
			  "metadata": {
			    "wiremock-gui": {
			      "folder": "/generated/Fry/get_courier_office"
			    }
			  }
			}`,
			path:           "/druz-fry/rpc",
			requestBody:    `{"jsonrpc":"2.0","method":"get_courier_office","params":{"coords":{"lat":"55.75","lon":"37.61"}},"id":"req-fry"}`,
			wantSubstrings: []string{"req-fry", "office-1"},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			id := createMapping(t, handler, tt.mapping)
			if tt.resetGRPC {
				reset := requestWithBody(t, handler, http.MethodPost, "/__admin/ext/grpc/reset", "")
				if reset.Code != http.StatusOK {
					t.Fatalf("grpc reset status = %d, want %d", reset.Code, http.StatusOK)
				}
			}

			matched := requestWithHeadersAndBody(
				t,
				handler,
				http.MethodPost,
				tt.path,
				map[string]string{"Content-Type": "application/json"},
				tt.requestBody,
			)
			if matched.Code != http.StatusOK {
				t.Fatalf("runtime status = %d, want %d: %s", matched.Code, http.StatusOK, matched.Body.String())
			}
			for _, want := range tt.wantSubstrings {
				if !strings.Contains(matched.Body.String(), want) {
					t.Fatalf("runtime body = %q, want containing %q", matched.Body.String(), want)
				}
			}

			deleted := requestWithBody(t, handler, http.MethodDelete, "/__admin/mappings/"+id, "")
			if deleted.Code != http.StatusOK {
				t.Fatalf("delete status = %d, want %d", deleted.Code, http.StatusOK)
			}

			afterDelete := requestWithHeadersAndBody(
				t,
				handler,
				http.MethodPost,
				tt.path,
				map[string]string{"Content-Type": "application/json"},
				tt.requestBody,
			)
			if afterDelete.Code != http.StatusNotFound {
				t.Fatalf("status after delete = %d, want %d", afterDelete.Code, http.StatusNotFound)
			}

			deletedAgain := requestWithBody(t, handler, http.MethodDelete, "/__admin/mappings/"+id, "")
			if deletedAgain.Code != http.StatusNotFound {
				t.Fatalf("second delete status = %d, want %d", deletedAgain.Code, http.StatusNotFound)
			}
		})
	}
}

func TestAutotestMappingLifecycle(t *testing.T) {
	handler := newTestHandler()
	createMapping(t, handler, `{
	  "name": "static reloadable mapping",
	  "persistent": true,
	  "request": {
	    "method": "POST",
	    "urlPath": "/static/reloadable"
	  },
	  "response": {
	    "status": 200,
	    "body": "before update"
	  },
	  "metadata": {
	    "wiremock-gui": {
	      "folder": "/static/reloadable"
	    }
	  }
	}`)

	list := requestWithBody(t, handler, http.MethodGet, "/__admin/mappings", "")
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", list.Code, http.StatusOK)
	}

	id := findMappingIDByNameAndFolder(t, decodeObjectResponse(t, list), "static reloadable mapping", "/static/reloadable")
	updated := requestWithBody(t, handler, http.MethodPut, "/__admin/mappings/"+id, `{
	  "name": "static reloadable mapping",
	  "persistent": true,
	  "request": {
	    "method": "POST",
	    "urlPath": "/static/reloadable"
	  },
	  "response": {
	    "status": 200,
	    "body": "after update"
	  },
	  "metadata": {
	    "wiremock-gui": {
	      "folder": "/static/reloadable"
	    }
	  }
	}`)
	if updated.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d: %s", updated.Code, http.StatusOK, updated.Body.String())
	}

	resp := requestWithBody(t, handler, http.MethodPost, "/static/reloadable", "")
	if resp.Code != http.StatusOK {
		t.Fatalf("runtime status = %d, want %d", resp.Code, http.StatusOK)
	}
	if resp.Body.String() != "after update" {
		t.Fatalf("runtime body = %q, want after update", resp.Body.String())
	}
}

func findMappingIDByNameAndFolder(t *testing.T, list map[string]any, name, folder string) string {
	t.Helper()

	mappings, ok := list["mappings"].([]any)
	if !ok {
		t.Fatalf("mappings = %T, want []any", list["mappings"])
	}
	for _, entry := range mappings {
		stub, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		if stub["name"] != name {
			continue
		}
		metadata, ok := stub["metadata"].(map[string]any)
		if !ok {
			continue
		}
		gui, ok := metadata["wiremock-gui"].(map[string]any)
		if !ok {
			continue
		}
		if gui["folder"] != folder {
			continue
		}
		id, ok := stub["id"].(string)
		if !ok || id == "" {
			t.Fatalf("matched mapping id = %v, want non-empty string", stub["id"])
		}
		return id
	}

	t.Fatalf("mapping %q in folder %q not found", name, folder)
	return ""
}
