import {Injectable} from 'angular2/core';
import {Http, Response} from 'angular2/http';
import {Observable} from 'rxjs/Observable';
import {NodeInfo} from '../models/nodeinfo';
import 'rxjs/Rx';
import {OlympusClient} from "./client";

@Injectable()
export class ApiClient implements OlympusClient {

  constructor(private http: Http) {}

  listNodes(id: string): Observable<NodeInfo[]> {
    return this.http.get(`/v1/ls/${id}`)
      .map((res: Response) => {
        var children = new Array<NodeInfo>();
        var json_array = res.json();
        for (var i = 0; i < json_array.length; i++) {
          children.push(<NodeInfo>json_array[i]);
        }
        return children;
    });
  }

  deleteNode(id: string): Observable<boolean> {
    return this.http.delete(`/v1/rm/${id}`)
      .map((res:Response) => res.status == 200);
  }

  handleError(error: Response) {
    console.error(error);
  }
}
