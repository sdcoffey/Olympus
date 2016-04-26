import {Observable} from "rxjs/Observable";
import {NodeInfo} from "../models/nodeinfo";

export interface OlympusClient {
  listNodes(id: string): Observable<NodeInfo[]>;
  deleteNode(id: string): Observable<boolean>;
}
