import {OlympusClient} from "./client";
import {Observable} from "rxjs/Observable";
import {NodeInfo} from "../models/nodeinfo";

export class FakeApiClient implements OlympusClient {
  
  listFiles():Observable<NodeInfo> {
    return undefined;
  }

  deleteFile(id:string):Observable<boolean> {
    return undefined;
  }
   
}
