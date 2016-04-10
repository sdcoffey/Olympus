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
  selectedIndex: number

  constructor(
    private _api: ApiClient,
    private _routeParams: RouteParams,
    private _router: Router
  ) {
    this.selectedIndex = -1
  }

  ngOnInit() {
    this.updateChildren(this._routeParams.get('parentId'));
  }

  updateChildren(id: string) {
    this.parentId = id;
    this._api.listFiles(this.parentId)
      .subscribe((children: any[]) => {
        this.children = new Array<NodeInfo>();
        for (var i = 0; i < children.length; i++) {
          this.children.push(new NodeInfo(
            children[i].Id,
            children[i].Name,
            children[i].Size,
            children[i].Mode
          ));
        }
      });
  }

  onChildSelected(index: number, fi: NodeInfo) {
    if (index == this.selectedIndex) {
      this.selectedIndex = -1;
    } else {
      this.selectedIndex = index;
    }
  }

  navigateTo(node: NodeInfo) {
    if (node.isDir()) {
      this._router.navigate(['/Browse', {parentId: node.Id}]);
    }
  }

  delete(event: MouseEvent, node: NodeInfo) {
    event.preventDefault();
    this._api.deleteNode(node.Id)
      .subscribe((success: boolean) => {
        if (success) {
          this.updateChildren(this.parentId);
        }
      });
  }
}
