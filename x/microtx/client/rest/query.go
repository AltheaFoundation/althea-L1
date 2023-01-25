package rest

// Example query handler which delegates the request to the ABCI Query method,
// allowing queries to fetch bytes directly from the store
// This method would also need to Unmarshal those bytes and respond in JSON
// func DataHandler(cliCtx client.Context, storeName string) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		vars := mux.Vars(r)
// 		key := vars["key"]

// 		res, height, err := cliCtx.Query(fmt.Sprintf("custom/%s/data/%s", storeName, key))
// 		if err != nil {
// 			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
// 			return
// 		}
// 		if len(res) == 0 {
// 			rest.WriteErrorResponse(w, http.StatusNotFound, "data not found")
// 			return
// 		}
//
// 		// e.g. querying Valsets
// 		var out types.Valset
// 		cliCtx.Codec.MustUnmarshalJSON(res, &out)
// 		rest.PostProcessResponse(w, cliCtx.WithHeight(height), res)
// 	}
// }
