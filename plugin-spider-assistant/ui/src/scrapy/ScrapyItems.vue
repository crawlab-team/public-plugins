<template>
  <cl-table
      class="scrapy-items"
      :data="tableData"
      :columns="tableColumns"
      hide-footer
  />
</template>

<script lang="ts">
import {computed, defineComponent, h} from 'vue';
import {ClTag} from 'crawlab-ui';
import {useRoute, useRouter} from 'vue-router';
import {useStore} from 'vuex';

const pluginName = 'spider-assistant';
const t = (path) => window['_tp'](pluginName, path);
const _t = window['_t'];

export default defineComponent({
  name: 'ScrapyItems',
  props: {
    form: {
      type: Object,
    }
  },
  setup(props, {emit}) {
    const router = useRouter();

    const route = useRoute();

    const id = computed(() => route.params.id);

    const store = useStore();

    const gotoFile = (filepath) => {
      store.commit(`spider/setDefaultFilePaths`, [filepath]);
      router.push({
        path: `/spiders/${id.value}/files`,
      });
    };

    const getType = ({type}) => {
      switch (type) {
        case 'str':
          return 'primary';
        case 'int':
        case 'float':
          return 'success';
        case 'bool':
          return 'danger';
        case 'dict':
        case 'list':
        case 'tuple':
          return 'warning';
        default:
          return 'info';
      }
    };

    const tableData = computed(() => {
      const {items} = props.form;
      return items || [];
    });

    const tableColumns = computed(() => {
      return [
        {
          key: 'name',
          label: t('scrapy.items.table.columns.name'),
          width: '200',
        },
        {
          key: 'fields',
          label: t('scrapy.items.table.columns.fields'),
          width: '800',
          value: (row) => {
            if (!row.fields) return [];
            return row.fields.map(f => h(ClTag, {
              label: f.name,
              clickable: true,
              type: getType(f),
              tooltip: f.type ? `${t('scrapy.items.fieldType')}: ${f.type}` : '',
            }));
          },
        },
        {
          key: 'actions',
          label: _t('components.table.columns.actions'),
          icon: ['fa', 'tools'],
          width: '180',
          fixed: 'right',
          buttons: (row) => [
            {
              type: 'primary',
              size: 'mini',
              icon: ['fa', 'search'],
              tooltip: _t('common.actions.view'),
              onClick: () => {
                gotoFile(row.filepath);
              }
            },
          ],
        },
      ];
    });

    return {
      tableData,
      tableColumns,
    };
  },
});
</script>

<style scoped>
.scrapy-items >>> .el-table {
  border-top: none;
  border-left: none;
  border-right: none;
}

.scrapy-items >>> .el-table .el-tag {
  margin-right: 10px;
}
</style>
