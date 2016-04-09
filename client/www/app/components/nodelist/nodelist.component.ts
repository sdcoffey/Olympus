import {Component, OnInit, Input} from 'angular2/core';
import {NodeInfo} from '../../models/nodeinfo';
import {ApiClient} from '../../services/apiclient';
import {Router, RouteParams} from 'angular2/router';

@Component({
  selector: 'node-list',
  templateUrl: 'app/components/nodelist/nodelist.html',
  styleUrls: ['app/components/nodelist/nodelist.css']
})
export class NodeListComponent implements OnInit {
  @Input() parentId: string
  children: NodeInfo[]

  constructor(
    private _api: ApiClient,
    private _routeParams: RouteParams,
    private _router: Router
  ) {}

  ngOnInit() {
    this.updateChildren(this._routeParams.get('parentId'));
  }

  updateChildren(id: string) {
    this.parentId = id;
    this._api.listFiles(this.parentId)
      .subscribe((children: NodeInfo[]) => {
        this.children = children;
      });
  }

  onChildSelected(fi: NodeInfo) {
    if (fi.isDir()) {
      this._router.navigate(['/Browse', {parentId: fi.Id}]);
    }
  }
}
