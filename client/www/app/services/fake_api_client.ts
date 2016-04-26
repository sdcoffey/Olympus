import {Observable} from "rxjs/Observable";
import {NodeInfo} from "../models/nodeinfo";
import {FAKE_DATA} from "./fakeData/fakedata";
import {Injectable} from "angular2/core";
import {Observer} from "rxjs/Observer";
import {ApiClient} from "./apiclient";

@Injectable()
export class FakeApiClient extends ApiClient {
  
  listNodes(id:string):Observable<NodeInfo[]> {
    return new Observable((observer: Observer<NodeInfo[]>) => {
      observer.next(<NodeInfo[]>FAKE_DATA[id]);
      observer.complete();
    })
  }

  deleteNode(id:string):Observable<boolean> {
    return undefined;
  }
}
