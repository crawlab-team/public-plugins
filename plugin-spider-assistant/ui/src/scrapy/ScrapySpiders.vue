<template>
  <cl-table
      class="scrapy-spiders"
      :data="tableData"
      :columns="tableColumns"
      hide-footer
  />
</template>

<script lang="ts">
import {computed, defineComponent, h} from 'vue';
import {ClNavLink} from 'crawlab-ui';
import {useRoute, useRouter} from 'vue-router';
import {useStore} from 'vuex';

const pluginName = 'spider-assistant';
const t = (path) => window['_tp'](pluginName, path);
const _t = window['_t'];

export default defineComponent({
  name: 'ScrapySpiders',
  props: {
    form: {
      type: Object,
      default: () => {
        return [];
      },
    },
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

    const tableData = computed(() => {
      const {spiders} = props.form;
      return spiders || [];
    });

    const tableColumns = computed(() => {
      return [
        {
          key: 'name',
          label: t('scrapy.spiders.table.columns.name'),
          icon: ['fa', 'font'],
          width: '160',
        },
        {
          key: 'type',
          label: t('scrapy.spiders.table.columns.type'),
          icon: ['fa', 'spider'],
          width: '160',
        },
        {
          key: 'filepath',
          label: t('scrapy.spiders.table.columns.filePath'),
          icon: ['fa', 'file'],
          width: '600',
          value: (row) => h(ClNavLink, {
            label: row.filepath,
            onClick: () => {
              gotoFile(row.filepath);
            }
          })
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
      t,
    };
  },
});
</script>

<style scoped>
.scrapy-spiders >>> .el-table {
  border-top: none;
  border-left: none;
  border-right: none;
}
</style>
