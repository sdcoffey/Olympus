import {Injectable} from 'angular2/core';
import {Http, Response} from 'angular2/http';
import {Observable} from 'rxjs/Observable';
import {NodeInfo} from '../models/nodeinfo';
import 'rxjs/Rx';
import {OlympusClient} from "./client";

@Injectable()
export class ApiClient extends OlympusClient {

  constructor(private http: Http) {}

  listFiles(id: string): Observable<NodeInfo[]> {
    return this.http.get(`/v1/ls/${id}`)
      .map((res:Response) => res.json());
  }

  deleteNode(id: string): Observable<boolean> {
    return this.http.delete(`/v1/rm/${id}`)
      .map((res:Response) => res.status == 200);
  }

  handleError(error: Response) {
    console.error(error);
  }
}
