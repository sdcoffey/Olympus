import {OlympusClient} from "./client";
import {Observable} from "rxjs/Observable";
import {NodeInfo} from "../models/nodeinfo";
import {FAKE_DATA} from "./fakeData/fakedata";
import {Injectable} from "angular2/core";
import {Observer} from "rxjs/Observer";

@Injectable()
export class FakeApiClient implements OlympusClient {

  listNodes(id:string):Observable<NodeInfo[]> {
    return new Observable((observer: Observer<NodeInfo[]>) => {
      if (id == "rootNode") {
        observer.next(FAKE_DATA.rootNode);
      } else {
        observer.next(FAKE_DATA.ghijkl);
      }
      observer.complete();
    })
  }

  deleteNode(id:string):Observable<boolean> {
    return undefined;
  }
}
