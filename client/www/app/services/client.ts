import {Observable} from "rxjs/Observable";
import {NodeInfo} from "../models/nodeinfo";

export interface OlympusClient {
  listFiles(): Observable<NodeInfo>;
  deleteFile(id: string): Observable<boolean>;
}
